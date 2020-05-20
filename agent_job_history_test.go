package gocdexporter

import (
	"testing"

	"github.com/pagero/go-gocd-ashwanth"
)

func TestGetJobHistory(t *testing.T) {
	a := &AgentJobHistory{}
	cache := AgentJobHistoryCache{}
	client := &MockClient{}
	agents, _ := client.GetAllAgents()

	err := a.GetJobHistory(client, agents, cache)
	if err != nil {
		t.Fatal(err)
	}

	//standard case
	jh1 := a.AgentJobHistory["fooagent"]
	if len(jh1) != 2 {
		t.Fatal("Expected job history for two agents")
	}
	if jh1[0].Name != "upload" {
		t.Fatal("Unexpected job name")
	}
	if jh1[1].Name != "upload" {
		t.Fatal("Unexpected job name")
	}
	if jh1[0].PipelineCounter != 2 {
		t.Fatalf("Expected pipeline counter to be '1', was: '%v'", jh1[0].PipelineCounter)
	}
	if jh1[1].PipelineCounter != 1 {
		t.Fatalf("Expected pipeline counter to be '1', was: '%v'", jh1[1].PipelineCounter)
	}

	//test empty job list
	if _, ok := a.AgentJobHistory["baragent"]; ok {
		t.Fatal("Expected baragent to not have any jobs")
	}

	//test already cached job for agent
	agent := &gocd.Agent{Hostname: "foobaragent", UUID: "111"}
	newAgents := []*gocd.Agent{agent}
	cache["foobaragent"] = getJobID(&JobHistory{&gocd.AgentJobHistory{
		Name:            "upload",
		PipelineName:    "distributions-all",
		PipelineCounter: 1,
		StageName:       "upload-installers",
		StageCounter:    "1",
	}})
	_ = a.GetJobHistory(client, newAgents, cache)
	jh3 := a.AgentJobHistory["foobaragent"]
	if len(jh3) != 1 {
		t.Fatal("Expected foobaragent to only have one job")
	}
	if jh3[0].PipelineCounter != 2 {
		t.Fatalf("Expected pipeline counter to be '2', was: '%v'", jh3[0].PipelineCounter)
	}
}

func TestGetScheduled(t *testing.T) {
	transition := &gocd.AgentJobStateTransition{
		StateChangeTime: "2020-05-20T00:20:56Z",
		State:           "Scheduled",
	}
	jh := &JobHistory{&gocd.AgentJobHistory{
		Name:                "upload",
		PipelineName:        "distributions-all",
		PipelineCounter:     1,
		StageName:           "upload-installers",
		StageCounter:        "1",
		JobStateTransitions: []gocd.AgentJobStateTransition{*transition},
	}}

	scheduled, err := jh.getScheduled()
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	expectedScheduledTimestamp := int64(1589934056)
	if scheduled != expectedScheduledTimestamp {
		t.Fatalf("Expected: %v to equal %v", scheduled, expectedScheduledTimestamp)
	}

	jh = &JobHistory{&gocd.AgentJobHistory{
		Name:            "upload",
		PipelineName:    "distributions-all",
		PipelineCounter: 1,
		StageName:       "upload-installers",
		StageCounter:    "1",
	}}
	_, err = jh.getScheduled()
	if err == nil {
		t.Fatalf("Expected error when no JobStateTransitions are present in JobHistory")
	}
}

func TestGetOrderedTransitions(t *testing.T) {
	a := &AgentJobHistory{}
	cache := AgentJobHistoryCache{}
	client := &MockClient{}
	agents, _ := client.GetAllAgents()

	_ = a.GetJobHistory(client, agents, cache)

	ajh := a.AgentJobHistory["bazagent"]
	st := ajh[0].GetOrderedStateTransitions()
	if st[0].State != "Scheduled" {
		t.Fatal("Transitions not sorted, expected 'Scheduled' got ", st[0].State)
	}
	if st[1].State != "Assigned" {
		t.Fatal("Transitions not sorted, expected 'Assigned' got ", st[1].State)
	}
	if st[2].State != "Preparing" {
		t.Fatal("Transitions not sorted, expected 'Preparing' got ", st[2].State)
	}
	if st[3].State != "Building" {
		t.Fatal("Transitions not sorted, expected 'Building' got ", st[3].State)
	}
	if st[4].State != "Rescheduled" {
		t.Fatal("Transitions not sorted, expected 'Rescheduled' got ", st[4].State)
	}
}
