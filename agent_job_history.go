package gocdexporter

import (
	"github.com/ashwanthkumar/go-gocd"
)

type AgentJobHistoryCache map[string]int

type AgentJobHistory struct {
	AgentJobHistory map[string][]*gocd.JobHistory
}

func (a *AgentJobHistory) Add(agent string, jobHistory *gocd.JobHistory) {
	if len(a.AgentJobHistory) == 0 {
		a.AgentJobHistory = make(map[string][]*gocd.JobHistory)
	}
	a.AgentJobHistory[agent] = append(a.AgentJobHistory[agent], jobHistory)
}

func (a *AgentJobHistory) GetJobHistory(client gocd.Client, agents []*gocd.Agent, cache AgentJobHistoryCache, maxPages int) error {
	for _, agent := range agents {
		offset := 0
		total := 1
		pageSize := 1
		firstRun := false
		for pageSize > 0 && offset/pageSize < maxPages && offset < total {
			cachedJobID := cache[agent.Hostname]
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
				cache[agent.Hostname] = jobs[0].ID
			}
			for _, job := range jobs {
				if cachedJobID >= job.ID && !firstRun {
					break
				}
				a.Add(agent.Hostname, job)
			}
			if !firstRun {
				break
			}
			offset += pageSize
		}
	}
	return nil
}
