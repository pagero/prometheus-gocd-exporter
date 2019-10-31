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

	err := a.GetJobHistory(client, agents, cache, 1)
	if err != nil {
		t.Fatal(err)
	}

	//standard case
	jh1 := a.AgentJobHistory["fooagent"]
	if len(jh1) != 2 {
		t.Fatal("Expected job history for two agents")
	}
	if jh1[0].ID != 2 {
		t.Fatal("Unexpected job id")
	}
	if jh1[1].ID != 1 {
		t.Fatal("Unexpected job id")
	}

	//test empty job list
	if _, ok := a.AgentJobHistory["baragent"]; ok {
		t.Fatal("Expected baragent to not have any jobs")
	}

	//test pagination
	jh2 := a.AgentJobHistory["bazagent"]
	if len(jh2) != 1 {
		t.Fatal("Expected bazagent to only have one job")
	}
	if jh2[0].ID != 2 {
		t.Fatal("Unexpected job id")
	}

	//test already cached job for agent
	agent := &gocd.Agent{Hostname: "foobaragent", UUID: "111"}
	newAgents := []*gocd.Agent{agent}
	cache["foobaragent"] = 1
	_ = a.GetJobHistory(client, newAgents, cache, 1)
	jh3 := a.AgentJobHistory["foobaragent"]
	if len(jh3) != 1 {
		t.Fatal("Expected foobaragent to only have one job")
	}
	if jh3[0].ID != 2 {
		t.Fatal("Unexpected job id")
	}
}

func TestGetOrderedTransitions(t *testing.T) {
	a := &AgentJobHistory{}
	cache := AgentJobHistoryCache{}
	client := &MockClient{}
	agents, _ := client.GetAllAgents()

	_ = a.GetJobHistory(client, agents, cache, 1)

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