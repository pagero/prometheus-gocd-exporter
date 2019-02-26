package gocdexporter

import (
	"github.com/ashwanthkumar/go-gocd"
	"github.com/prometheus/client_golang/prometheus"
)

type AgentJobHistoryCache struct {
	AgentJob map[string]int
}

func NewAgentJobHistoryCache() *AgentJobHistoryCache {
	return &AgentJobHistoryCache{}
}

func (c *AgentJobHistoryCache) Get(agentHostname string) int {
	if job, ok := c.AgentJob[agentHostname]; ok {
		return job
	}
	return 0
}

func (c *AgentJobHistoryCache) Set(agentHostname string, job int) {
	if len(c.AgentJob) == 0 {
		c.AgentJob = make(map[string]int)
	}
	c.AgentJob[agentHostname] = job
}

func agentJobHistory(client gocd.Client, agents []*gocd.Agent, agentJobResultCounter *prometheus.CounterVec, cache *AgentJobHistoryCache, maxPages int) error {
	for _, agent := range agents {
		offset := 0
		total := 1
		pageSize := 1
		firstRun := false
		for offset/pageSize < maxPages && offset < total {
			cachedJobID := cache.Get(agent.Hostname)
			if cachedJobID == 0 {
				firstRun = true
			}
			history, err := client.AgentRunJobHistory(agent.UUID, offset)
			if err != nil {
				return err
			}
			jobs := history.Jobs
			pageSize = history.Pagination.PageSize
			total = history.Pagination.Total
			if len(jobs) > 0 && jobs[0].ID > cachedJobID {
				cache.Set(agent.Hostname, jobs[0].ID)
			}
			for _, job := range jobs {
				if cachedJobID >= job.ID && !firstRun {
					break
				}
				agentJobResultCounter.WithLabelValues(
					agent.Hostname, job.PipelineName, job.StageName, job.Name, job.Result,
				).Inc()
			}
			if !firstRun {
				break
			}
			offset += pageSize
		}
	}
	return nil
}
