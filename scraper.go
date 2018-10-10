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
	jobsByState, jobsByStateScrape := newJobsByStateCollector(conf, ccCache)
	if err := conf.Registerer.Register(jobsByState); err != nil {
		return nil, err
	}

	return func(ctx context.Context) error {
		if err := ccCache.Update(ctx); err != nil {
			return err
		}
		routines := 2
		errCh := make(chan error, routines)
		go func() { errCh <- agentScrape(ctx) }()
		go func() { errCh <- jobsByStateScrape(ctx) }()

		for n := 0; n < routines; n++ {
			if err := <-errCh; err != nil {
				return err
			}
		}

		return nil
	}, nil
}

func newJobsByStateCollector(conf *Config, ccCache *CCTrayCache) (*prometheus.GaugeVec, Scraper) {
	gauge := prometheus.NewGaugeVec(
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

	return gauge, func(ctx context.Context) error {
		cc, err := ccCache.Get(ctx)
		if err != nil {
			return err
		}
		jobStates := map[string]map[string]int{}
		for _, project := range cc.Projects {
			if project.Activity == "Sleeping" {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			log.Printf("Project: %s - %s\n", project.Name, project.URL)
			p, err := client.GetPipelineInstance(project.Pipeline(), int(project.Instance()))
			if err != nil {
				return err
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
			log.Printf("\tStages: %d - Jobs %d\n", stages, jobs)
		}

		gauge.Reset()
		for pipeline, states := range jobStates {
			for state, count := range states {
				gauge.WithLabelValues(
					state, pipeline,
				).Set(float64(count))
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
