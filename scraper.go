package gocdexporter

import (
	"context"
	"log"

	"github.com/ashwanthkumar/go-gocd"
	"github.com/prometheus/client_golang/prometheus"
)

// Config for the Scraper.
type Config struct {
	Namespace  string
	Registerer prometheus.Registerer
	GocdURL    string
	GocdUser   string
	GocdPass   string
}

// Scraper runs one scrape loop for collecting metrics.
type Scraper func(context.Context) error

// NewScraper configures prometheus metrics to be scraped from GoCD.
func NewScraper(conf *Config) (Scraper, error) {
	agents, agentScrape := newAgentCollector(conf)
	if err := conf.Registerer.Register(agents); err != nil {
		return nil, err
	}

	ccCache := NewCCTrayCache(conf.GocdURL, conf.GocdUser, conf.GocdPass)
	jobsByState, pipelineInstanceScrape := newPipelineInstanceCollector(conf, ccCache)
	if err := conf.Registerer.Register(jobsByState); err != nil {
		return nil, err
	}

	pipelineResult, flappingResult, buildsCount, pipelineResultScrape := newPipelineResultCollector(conf, ccCache)
	if err := conf.Registerer.Register(pipelineResult); err != nil {
		return nil, err
	}
	if err := conf.Registerer.Register(buildsCount); err != nil {
		return nil, err
	}
	if err := conf.Registerer.Register(flappingResult); err != nil {
		return nil, err
	}

	scrapers := []Scraper{
		agentScrape,
		pipelineInstanceScrape,
		pipelineResultScrape,
	}

	return func(ctx context.Context) error {
		if err := ccCache.Update(ctx); err != nil {
			return err
		}
		errCh := make(chan error, len(scrapers))
		for _, scraper := range scrapers {
			go func(s Scraper) { errCh <- s(ctx) }(scraper)
		}

		for range scrapers {
			if err := <-errCh; err != nil {
				return err
			}
		}

		return nil
	}, nil
}

func newPipelineInstanceCollector(conf *Config, ccCache *CCTrayCache) (
	jobsByState *prometheus.GaugeVec, _ Scraper,
) {
	jobsByState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "jobs_by_state_count",
			Help:      "Number of jobs",
		},
		[]string{
			// "Scheduled", "Assigned", "Preparing",
			// "Building", "Completing", "Completed"
			"state",
			"pipeline",
		},
	)

	client := gocd.New(conf.GocdURL, conf.GocdUser, conf.GocdPass)

	return jobsByState, func(ctx context.Context) error {
		cc, err := ccCache.Get(ctx)
		if err != nil {
			return err
		}
		jobStates := map[string]map[string]int{}
		pipelineCache := map[string]*gocd.PipelineInstance{}
		for _, project := range cc.Projects {
			if project.Activity == "Sleeping" {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			log.Printf("jobs_by_state: Project: %s - %s\n", project.Name, project.URL)
			// Pipeline occurs once for each stage, cache to avoid unnecessary API calls.
			p, ok := pipelineCache[project.Pipeline()]
			if !ok {
				var err error
				p, err = client.GetPipelineInstance(project.Pipeline(), int(project.Instance()))
				if err != nil {
					return err
				}
				pipelineCache[project.Pipeline()] = p
			}
			stages := len(p.Stages)
			jobs := 0
			jobStates[project.Pipeline()] = map[string]int{
				"Scheduled":  0,
				"Assigned":   0,
				"Preparing":  0,
				"Building":   0,
				"Completing": 0,
				"Completed":  0,
			}
			for _, stage := range p.Stages {
				for _, job := range stage.Jobs {
					jobs++
					jobStates[project.Pipeline()][job.State]++
				}
			}
			log.Printf("jobs_by_state:\tStages: %d - Jobs %d\n", stages, jobs)
		}

		jobsByState.Reset()
		for pipeline, states := range jobStates {
			for state, count := range states {
				jobsByState.WithLabelValues(
					state, pipeline,
				).Set(float64(count))
			}
		}

		return nil
	}
}

func newPipelineResultCollector(conf *Config, ccCache *CCTrayCache) (
	pipelineResultGauge *prometheus.GaugeVec, flappingResultGauge *prometheus.GaugeVec,
	buildsCount *prometheus.CounterVec, _ Scraper,
) {
	pipelineResultGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "pipelines_result",
			Help:      "Pipeline result statuses",
		},
		[]string{
			"pipeline",
			"stage",
			"result",
		},
	)
	// Can be used to detect changes in results a.k.a. flapping pipeline.
	flappingResultGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "pipelines_result_flapping",
			Help:      "Pipeline result statuses as numbers",
		},
		[]string{
			"pipeline",
			"stage",
		},
	)
	buildsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: conf.Namespace,
			Name:      "builds_count",
			Help:      "Number of builds",
		},
		[]string{
			"pipeline",
		},
	)

	return pipelineResultGauge, flappingResultGauge, buildsCount, func(ctx context.Context) error {
		cc, err := ccCache.Get(ctx)
		if err != nil {
			return err
		}

		pipelineResults := map[string]map[string]string{}
		for _, project := range cc.Projects {
			if project.LastResult == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			log.Printf("pipelines_result: Project: %s - %s\n", project.Name, project.URL)

			r, ok := pipelineResults[project.Pipeline()]
			if !ok {
				r = map[string]string{}
				buildsCount.WithLabelValues(
					project.Pipeline(),
				).Set(float64(project.Instance()))
			}
			r[project.Stage()] = project.LastResult
			pipelineResults[project.Pipeline()] = r
			log.Printf("pipelines_result:\tStage: %s - Result: %s\n", project.Stage(), project.LastResult)
		}

		pipelineResultGauge.Reset()
		flappingResultGauge.Reset()
		for pipeline, stages := range pipelineResults {
			for stage, result := range stages {
				pipelineResultGauge.WithLabelValues(
					pipeline, stage, result,
				).Set(1)
				resultAsValue := 0.0
				if result == "Success" {
					resultAsValue = 1.0
				}
				flappingResultGauge.WithLabelValues(
					pipeline, stage,
				).Set(resultAsValue)
			}
		}

		return nil
	}
}

func newAgentCollector(conf *Config) (*prometheus.GaugeVec, Scraper) {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "agent_count",
			Help:      "Number of agents",
		},
		[]string{
			// Idle, Building, Cancelled, Unknown.
			"build_state",
			// Building, LostContact, Missing, Unknown.
			"agent_state",
			// Pending, Enabled, Disabled.
			"agent_config_state",
		},
	)

	client := gocd.New(conf.GocdURL, conf.GocdUser, conf.GocdPass)
	return gauge, func(ctx context.Context) error {
		agents, err := client.GetAllAgents()
		if err != nil {
			log.Fatal(err)
		}
		gauge.Reset()
		for _, a := range agents {
			gauge.WithLabelValues(
				a.BuildState, a.AgentState, a.AgentConfigState,
			).Add(1)
		}

		return nil
	}
}
