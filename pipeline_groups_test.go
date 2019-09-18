package gocdexporter

import (
	"testing"
	"time"
)

func TestGetPipelineGroups(t *testing.T) {
	pg := PipelineGroups{}
	client := NewMockClient()

	// test standard get case
	pGroup, err := pg.GetPipelineGroup(client, "Pipeline 1")
	if err != nil {
		t.Fatal(err)
	}
	if pGroup != "PipelineGroup 1" {
		t.Fatal("Unexpected pipeline group")
	}

	pGroup, _ = pg.GetPipelineGroup(client, "Pipeline 2")
	if pGroup != "Unknown" {
		t.Fatal("Unexpected pipeline group")
	}
}

func TestGetPipelineGroupsCache(t *testing.T) {
	pg := PipelineGroups{}
	client := NewMockClient()

	pGroup, _ := pg.GetPipelineGroup(client, "Pipeline 1")
	if pGroup != "PipelineGroup 1" {
		t.Fatal("Unexpected pipeline group: ", pGroup)
	}

	client.pg[0].Name = "PipelineGroup 2"

	pGroup, _ = pg.GetPipelineGroup(client, "Pipeline 1")
	if pGroup != "PipelineGroup 1" {
		t.Fatal("Unexpected pipeline group: ", pGroup)
	}

	pg.lastUpdated = time.Now().Add(-time.Hour * 6)

	pGroup, _ = pg.GetPipelineGroup(client, "Pipeline 1")
	if pGroup != "PipelineGroup 2" {
		t.Fatal("Unexpected pipeline group: ", pGroup)
	}
}
