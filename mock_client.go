package gocdexporter

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"github.com/ashwanthkumar/go-gocd"
)

type MockClient struct {
	pg []*gocd.PipelineGroup
}

func NewMockClient() *MockClient {
	mc := &MockClient{}
	f, _ := ioutil.ReadFile("fixtures/get_pipeline_groups.json")
	_ = json.Unmarshal([]byte(string(f)), &mc.pg)

	return mc
}

func (c *MockClient) GetAllAgents() ([]*gocd.Agent, error) {
	f, _ := ioutil.ReadFile("fixtures/get_all_agents.json")
	type EmbeddedObj struct {
		Agents []*gocd.Agent `json:"agents"`
	}
	responseFormat := struct {
		Embedded EmbeddedObj `json:"_embedded"`
	}{}

	_ = json.Unmarshal([]byte(string(f)), &responseFormat)

	return responseFormat.Embedded.Agents, nil
}
func (c *MockClient) GetAgent(uuid string) (*gocd.Agent, error) {
	return nil, nil
}
func (c *MockClient) UpdateAgent(uuid string, agent *gocd.Agent) (*gocd.Agent, error) {
	return nil, nil
}
func (c *MockClient) DisableAgent(uuid string) error {
	return nil
}
func (c *MockClient) EnableAgent(uuid string) error {
	return nil
}
func (c *MockClient) DeleteAgent(uuid string) error {
	return nil
}
func (c *MockClient) AgentRunJobHistory(uuid string, offset int) (*gocd.JobRunHistory, error) {
	f, _ := ioutil.ReadFile("fixtures/" + uuid + "_agent_job_history_" + strconv.Itoa(offset) + ".json")
	h := new(gocd.JobRunHistory)
	_ = json.Unmarshal([]byte(string(f)), &h)
	return h, nil
}

func (c *MockClient) GetPipelineGroups() ([]*gocd.PipelineGroup, error) {
	return c.pg, nil
}
func (c *MockClient) GetPipelineInstance(string, int) (*gocd.PipelineInstance, error) {
	return nil, nil
}
func (c *MockClient) GetPipelineHistoryPage(string, int) (*gocd.PipelineHistoryPage, error) {
	return nil, nil
}
func (c *MockClient) GetPipelineStatus(string) (*gocd.PipelineStatus, error) {
	return nil, nil
}
func (c *MockClient) PausePipeline(string, string) (*gocd.SimpleMessage, error) {
	return nil, nil
}
func (c *MockClient) UnpausePipeline(string) (*gocd.SimpleMessage, error) {
	return nil, nil
}
func (c *MockClient) UnlockPipeline(string) (*gocd.SimpleMessage, error) {
	return nil, nil
}
func (c *MockClient) GetScheduledJobs() ([]*gocd.ScheduledJob, error) {
	return nil, nil
}
func (c *MockClient) GetJobHistory(pipeline, stage, job string, offset int) ([]*gocd.JobHistory, error) {
	return nil, nil
}
func (c *MockClient) GetAllEnvironmentConfigs() ([]*gocd.EnvironmentConfig, error) {
	return nil, nil
}
func (c *MockClient) GetEnvironmentConfig(name string) (*gocd.EnvironmentConfig, error) {
	return nil, nil
}
