package gocdexporter

import (
	"time"

	"github.com/ashwanthkumar/go-gocd"
)

type PipelineGroups struct {
	pipelineGroups map[string]string
	lastUpdated    time.Time
}

func (pg *PipelineGroups) GetPipelineGroup(client gocd.Client, pipeline string) (string, error) {
	if pg.lastUpdated.IsZero() || time.Now().Sub(pg.lastUpdated) > time.Hour {
		err := pg.updatePipelineGroups(client)
		if err != nil {
			return "", err
		}
	}

	if _, ok := pg.pipelineGroups[pipeline]; !ok {
		return "Unknown", nil
	}

	return pg.pipelineGroups[pipeline], nil
}

func (pg *PipelineGroups) updatePipelineGroups(client gocd.Client) error {
	pgs, err := client.GetPipelineGroups()
	if err != nil {
		return err
	}

	pg.pipelineGroups = make(map[string]string)
	for _, pGroup := range pgs {
		for _, p := range pGroup.Pipelines {
			pg.pipelineGroups[p.Name] = pGroup.Name
		}
	}

	pg.lastUpdated = time.Now()

	return nil
}
