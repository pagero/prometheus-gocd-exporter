package gocdexporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"testing"
)

func TestAgentJobHistoryStats(t *testing.T) {
	agentJobCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agent_job_results",
		},
		[]string{"agent", "pipeline", "stage", "job", "result"},
	)
	cache := NewAgentJobHistoryCache()
	client := &MockClient{}
	agents, _ := client.GetAllAgents()
	err := agentJobHistory(client, agents, agentJobCounter, cache, 2)
	if err != nil {
		t.Fatal(err)
	}
	if cache.Get("fooagent") != 2 {
		t.Fatal("expected last job in cache to be 2")
	}
	if cache.Get("baragent") != 0 {
		t.Fatal("expected last job in cache to be 0")
	}
	if cache.Get("bazagent") != 2 {
		t.Fatal("expected last job in cache to be 2")
	}
}

func TestCache(t *testing.T) {
	cache := NewAgentJobHistoryCache()
	cache.Set("fooagent", 1)
	job := cache.Get("fooagent")
	if job != 1 {
		t.Fatal("expected cached job")
	}
	miss := cache.Get("baragent")
	if miss != 0 {
		t.Fatal("expected no cache hit")
	}
}
