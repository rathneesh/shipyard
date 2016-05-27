package model

import (
	"time"
)

const (
	BuildStartActionLabel = "start"

	BuildStatusNewLabel = "new"
	BuildStatusRunningLabel = "running"
	BuildStatusStoppedLabel = "stopped"
	BuildStatusFinishedSuccess = "finished_success"
	BuildStatusFinishedFailed = "finished_failed"
)

type Build struct {
	ID        string         `json:"id,omitempty" gorethink:"id,omitempty"`
	StartTime time.Time      `json:"startTime,omitempty" gorethink:"startTime,omitempty"`
	EndTime   time.Time      `json:"endTime,omitempty" gorethink:"endTime,omitempty"`
	Config    *BuildConfig   `json:"config,omitempty" gorethink:"config,omitempty"`
	Status    *BuildStatus   `json:"status,omitempty" gorethink:"status,omitempty"`
	Results   []*BuildResult `json:"results" gorethink:"results"`
	TestId    string         `json:"testId" gorethink:"testId"`
	ProjectId string         `json:"projectId" gorethink:"projectId"`
}

func NewBuild(config *BuildConfig, status *BuildStatus, results []*BuildResult, testId string, projectId string) *Build {

	return &Build{
		Config:    config,
		Status:    status,
		Results:   results,
		TestId:    testId,
	}
}

type BuildConfig struct {
	ID               string            `json:"-" gorethink:"id,omitempty"`
	Name             string            `json:"name" gorethink:"name"`
	Description      string            `json:"description" gorethink:"description"`
	Targets          []*TargetArtifact `json:"targets" gorethink:"targets"`
	SelectedTestType string            `json:"selectedTestType" gorethink:"selectedTestType"`
	ProviderId       string            `json:"providerId" gorethink:"providerId"`
}

func (b *BuildConfig) NewBuildConfig(name string, description string, targets []*TargetArtifact, selectedTestType string, providerId string) *BuildConfig {

	return &BuildConfig{
		Name:             name,
		Description:      description,
		Targets:          targets,
		SelectedTestType: selectedTestType,
		ProviderId:       providerId,
	}
}

type BuildResult struct {
	ID             string          `json:"-" gorethink:"id,omitempty"`
	BuildId        string          `json:"buildId" gorethink:"buildId"`
	TargetArtifact *TargetArtifact `json:"targetArtifact" gorethink:"targetArtifact"`
	ResultEntries  []string        `json:"resultEntries" gorethink:"resultEntries"`
	TimeStamp      time.Time       `json:"-" gorethink:"timeStamp,omitempty"`
	Successful     bool            `json:"successful" gorethink:"successful"`
}

//type ResultEntry string

func NewBuildResult(buildId string, artifact *TargetArtifact, results []string, successful bool) *BuildResult {

	return &BuildResult{
		BuildId:        buildId,
		TargetArtifact: artifact,
		ResultEntries:  results,
		TimeStamp:      time.Now(),
		Successful:     successful,
	}
}

type BuildStatus struct {
	ID      string `json:"-" gorethink:"id,omitempty"`
	BuildId string `json:"buildId" gorethink:"buildId"`
	Status  string `json:"status" gorethink:"status"`
}

func (b *BuildStatus) NewBuildStatus(buildId string, status string) *BuildStatus {

	return &BuildStatus{
		BuildId: buildId,
		Status:  status,
	}
}

type BuildAction struct {
	ID     string `json:"-" gorethink:"id,omitempty"`
	Action string `json:"action" gorethink:"action"`
}

func (b *BuildStatus) NewBuildAction(action string) *BuildAction {

	return &BuildAction{
		Action: action,
	}
}
