package gocdexporter

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

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
	ccCache := NewCCTrayCache(conf.GocdURL, conf.GocdUser, conf.GocdPass)

	scrapers := []Scraper{}
	add := func(c []prometheus.Collector, s Scraper) error {
		scrapers = append(scrapers, s)
		for _, collector := range c {
			if err := conf.Registerer.Register(collector); err != nil {
				return err
			}
		}
		return nil
	}

	if err := add(newAgentCollector(conf)); err != nil {
		return nil, err
	}
	if err := add(newScheduledCollector(conf)); err != nil {
		return nil, err
	}
	if err := add(newPipelineResultCollector(conf, ccCache)); err != nil {
		return nil, err
	}
	if err := add(newPipelineDurationCollector(conf, ccCache)); err != nil {
		return nil, err
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

func newScheduledCollector(conf *Config) (
	[]prometheus.Collector, Scraper,
) {
	scheduledGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "jobs_scheduled_count",
			Help:      "Number of jobs scheduled",
		},
		[]string{
			"pipeline",
			"stage",
			"job",
		},
	)

	client := gocd.New(conf.GocdURL, conf.GocdUser, conf.GocdPass)

	return []prometheus.Collector{scheduledGauge}, func(ctx context.Context) error {
		jobs, err := client.GetScheduledJobs()
		if err != nil {
			return err
		}
		scheduledGauge.Reset()
		for _, job := range jobs {
			parts := strings.Split(job.BuildLocator, "/")
			if len(parts) != 5 {
				return errors.New("scheduledCollector: unexpected scheduled build locator")
			}
			scheduledGauge.WithLabelValues(
				parts[0], parts[2], parts[4],
			).Set(1)
		}

		return nil
	}
}

func newPipelineResultCollector(conf *Config, ccCache *CCTrayCache) (
	[]prometheus.Collector, Scraper,
) {
	pipelineResultGauge := prometheus.NewGaugeVec(
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
	flappingResultGauge := prometheus.NewGaugeVec(
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
	buildsCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: conf.Namespace,
			Name:      "builds_count",
			Help:      "Number of builds",
		},
		[]string{
			"pipeline",
		},
	)
	// Cache to check if counter goes up or down. Sometimes and older instance count
	// is displayed in cctray.
	buildsCountCache := map[string]int64{}
	collectors := []prometheus.Collector{pipelineResultGauge, flappingResultGauge, buildsCount}

	return collectors, func(ctx context.Context) error {
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
			}
			r[project.Stage()] = project.LastResult
			pipelineResults[project.Pipeline()] = r

			c, ok := buildsCountCache[project.Pipeline()]
			if !ok {
				buildsCountCache[project.Pipeline()] = project.Instance()
			}
			// Make sure we don't try to decrease a counter because of old instance info.
			if c < project.Instance() {
				buildsCount.WithLabelValues(
					project.Pipeline(),
				).Set(float64(project.Instance()))
			}

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

func newPipelineDurationCollector(conf *Config, ccCache *CCTrayCache) (
	[]prometheus.Collector, Scraper,
) {
	pipelineStateGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "pipelines_by_state_count",
			Help:      "Pipeline state",
		},
		[]string{
			"pipeline",
			"state",
		},
	)
	pipelineDurationGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "pipelines_by_duration_seconds",
			Help:      "Pipeline activity duration",
		},
		[]string{
			"pipeline",
			"stage",
		},
	)
	collectors := []prometheus.Collector{pipelineStateGauge, pipelineDurationGauge}

	activityStarted := map[string]map[string]time.Time{}
	return collectors, func(ctx context.Context) error {
		cc, err := ccCache.Get(ctx)
		if err != nil {
			return err
		}

		pipelineStates := map[string]string{}
		for _, project := range cc.Projects {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			pipelineStates[project.Pipeline()] = project.Activity

			stages, ok := activityStarted[project.Pipeline()]
			if !ok {
				stages = map[string]time.Time{}
			}
			// Track execution duration
			switch project.Activity {
			case "Building":
				if _, ok := stages[project.Stage()]; !ok {
					stages[project.Stage()] = time.Now()
				}
			case "Sleeping":
				started, ok := stages[project.Stage()]
				if !ok {
					break
				}
				pipelineDurationGauge.WithLabelValues(
					project.Pipeline(), project.Stage(),
				).Set(time.Since(started).Seconds())
				delete(stages, project.Stage())
			}
			activityStarted[project.Pipeline()] = stages
		}

		pipelineStateGauge.Reset()
		for pipeline, state := range pipelineStates {
			pipelineStateGauge.WithLabelValues(
				pipeline, state,
			).Set(1)
		}

		return nil
	}
}

func newAgentCollector(conf *Config) ([]prometheus.Collector, Scraper) {
	agentCountGauge := prometheus.NewGaugeVec(
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
	agentJobGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: conf.Namespace,
			Name:      "agent_job",
			Help:      "Assigned jobs",
		},
		[]string{
			"pipeline",
			"stage",
			"job",
			"rerun",
			"state",
			"result",
			"agent",
		},
	)

	client := gocd.New(conf.GocdURL, conf.GocdUser, conf.GocdPass)
	return []prometheus.Collector{agentCountGauge, agentJobGauge}, func(ctx context.Context) error {
		agents, err := client.GetAllAgents()
		if err != nil {
			return err
		}
		// Quick stats first
		agentCountGauge.Reset()
		for _, a := range agents {
			agentCountGauge.WithLabelValues(
				a.BuildState, a.AgentState, a.AgentConfigState,
			).Add(1)
		}

		// Slower scrape for each job history
		jobStats := [][]string{}
		for _, a := range agents {
			if a.BuildState != "Building" {
				continue
			}
			history, err := client.GetJobHistory(a.BuildDetails.PipelineName, a.BuildDetails.StageName, a.BuildDetails.JobName, 0)
			if err != nil {
				return err
			}
			if len(history) == 0 {
				return errors.New("AgentCollector: no history result")
			}
			job := history[0]
			if job.AgentUUID != a.UUID {
				log.Println("AgentCollector: mismatched UUID in job history")
			}
			rerun := "no"
			if job.ReRun {
				rerun = "yes"
			}
			jobStats = append(jobStats, []string{
				job.PipelineName, job.StageName, job.Name,
				rerun, job.State, job.Result, a.Hostname,
			})
		}
		agentJobGauge.Reset()
		for _, stats := range jobStats {
			agentJobGauge.WithLabelValues(stats...).Set(1)
		}

		return nil
	}
}
