package gocdexporter

import (
	"fmt"
	"github.com/pagero/go-gocd-ashwanth"
	"sort"
	"time"
)

type AgentJobHistoryCache map[string]string

type AgentJobHistory struct {
	AgentJobHistory map[string][]*JobHistory
}

type JobHistory struct {
	*gocd.AgentJobHistory
}

func (j *JobHistory) GetOrderedStateTransitions() []gocd.AgentJobStateTransition {
	t := j.JobStateTransitions
	sort.Slice(t, func(i, j int) bool {
		t1, err := time.Parse(time.RFC3339, t[i].StateChangeTime)
		if err != nil {
			return false
		}
		t2, err := time.Parse(time.RFC3339, t[j].StateChangeTime)
		if err != nil {
			return false
		}
		return t1.Unix() < t2.Unix()
	})
	return t
}

func (a *AgentJobHistory) Add(agent string, jobHistory *JobHistory) {
	if len(a.AgentJobHistory) == 0 {
		a.AgentJobHistory = make(map[string][]*JobHistory)
	}
	a.AgentJobHistory[agent] = append(a.AgentJobHistory[agent], jobHistory)
}

func (a *AgentJobHistory) GetJobHistory(client gocd.Client, agents []*gocd.Agent, cache AgentJobHistoryCache) error {
	offset := 0
	pageSize := 100
	for _, agent := range agents {
		firstRun := false
		cachedJobID := cache[agent.Hostname]
		if cachedJobID == "" {
			firstRun = true
		}
		history, err := client.AgentRunJobHistory(agent.UUID, offset, pageSize)
		if err != nil {
			return err
		}
		jobs := history.Jobs
		firstID := ""
		if len(jobs) > 0 {
			firstID = getJobID(&JobHistory{jobs[0]})
			if firstID != cachedJobID {
				cache[agent.Hostname] = firstID
			}
		}
		for _, job := range jobs {
			jh := &JobHistory{job}
			if cachedJobID == getJobID(jh) && !firstRun {
				break
			}
			a.Add(agent.Hostname, jh)
		}
	}
	return nil
}

func getJobID(jh *JobHistory) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", jh.PipelineName, jh.PipelineCounter, jh.StageName, jh.StageCounter, jh.Name)
}

func (ajh *JobHistory) getScheduled() (int64, error) {
	for _, t := range ajh.JobStateTransitions {
		if t.State == "Scheduled" {
			timestamp, err := time.Parse(time.RFC3339, t.StateChangeTime)
			if err != nil {
				return 0, err
			}
			return timestamp.Unix(), nil
		}
	}
	return 0, fmt.Errorf("Could not find Scheduled time for job with id: %v", getJobID(ajh))
}
