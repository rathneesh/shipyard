package manager

import (
	"encoding/json"
	r "github.com/dancannon/gorethink"
	"github.com/shipyard/shipyard/model"
	log "github.com/Sirupsen/logrus"
	"bytes"
	"strings"
	"net"
)

const (
	database        = "shipyard"
)

var session *r.Session

type CollectedData struct {
	MAC          string
	Username     string
	Images       []Image
	Projects     []Project
	Builds       []Build
	Registries   []Registry
	Tests        []Test
	Results      []BuildResult
}

type Build struct {
	Id        string `json:"id,omitempty" gorethink:"id,omitempty"`
	ProjectId string `json:"projectId" gorethink:"projectId"`
	TestId    string `json:"testId" gorethink:"testId"`
	StartTime string `json:"startTime,omitempty" gorethink:"startTime,omitempty"`
	Status    Status `json:"status,omitempty" gorethink:"status,omitempty"`
}

type Status struct {
	Status string `json:"status" gorethink:"status"`
}

type Results struct {
	ResultEntries []string
}

type Project struct {
	Id           string `json:"id,omitempty" gorethink:"id,omitempty"`
	Name         string `json:"name" gorethink:"name"`
	CreationTime string `json:"creationTime" gorethink:"creationTime"`
	Status       string `json:"status" gorethink:"status"`
	ImageIds     []string `json:"imageids" gorethink:"imageids"`
	TestIds      []string `json:"testids" gorethink:"testids"`
	Images       []Image `json:"images,omitempty" gorethink:"-"`
	Tests        []Test `json:"tests,omitempty" gorethink:"-"`
}

type Image struct {
	Id             string `json:"id,omitempty" gorethink:"id,omitempty"`
	Name           string `json:"name" gorethink:"name"`
	ImageId        string `json:"imageId" gorethink:"imageId"`
	Description    string `json:"description" gorethink:"description"`
	RegistryId     string `json:"registryId" gorethink:"registryId"`
	Tag            string `json:"tag" gorethink:"tag"`
	IlmTags        []string `json:"ilmTags" gorethink:"ilmTags"`
	Location       string `json:"location" gorethink:"location"`
	SkipImageBuild bool `json:"skipImageBuild" gorethink:"skipImageBuild"`
}

type Repository struct {
	Name         string
	Tag          string
	FsLayers     []FsLayer
	Signatures   []Signature
	HasProblems  bool
	Message      string
	RegistryUrl  string
	RegistryName string
}

type FsLayer struct {
	BlobSum string
}
type Signature struct {
	Header    Header
	Signature string
	Protected string
}

type Header struct {
	Algorithm string
}
type Registry struct {
	Id   string `json:"id,omitempty" gorethink:"id,omitempty"`
	Name string `json:"name,omitempty" gorethink:"name,omitempty"`
	Addr string `json:"addr,omitempty" gorethink:"addr,omitempty"`
}
type Test struct {
	Id       string `json:"id,omitempty" gorethink:"id,omitempty"`
	Name     string `json:"name" gorethink:"name"`
	Provider Provider `json:"provider" gorethink:"provider"`
}
type Provider struct {
	ProviderType string `json:"providerType" gorethink:"providerType"`
}
type BuildResult struct {
	Id             string `json:"id,omitempty" gorethink:"id,omitempty"`
	BuildId        string `json:"buildId" gorethink:"buildId"`
	ResultEntries  []string `json:"resultEntries" gorethink:"resultEntries"`
	TargetArtifact TargetArtifact `json:"targetArtifact" gorethink:"targetArtifact"`
}
type TargetArtifact struct {
	Artifact Artifact `json:"artifact" gorethink:"artifact"`
}

type Artifact struct {
	ImageId string
	Link    string
}

func initDatabase(addr string) {
	var err error
	session, err = r.Connect(r.ConnectOpts{
		Address:  addr,
		Database: database,
	})

	if err != nil {
		log.Error(err)
		return
	}
}

func retrieveMACAddress() string {
	id := "anon"
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Name != "lo" {
				hw := iface.HardwareAddr.String()
				id = strings.Replace(hw, ":", "", -1)
				break
			}
		}
	}
	return id
}

func retrieveAllProjects() []Project {
	result := []Project{}
	rows, err := r.Table(tblNameProjects).Run(session)
	if err != nil {
		log.Error(err)
		return []Project{}
	}
	projects := []*model.Project{}
	err2 := rows.All(&projects)
	if err2 != nil {
		log.Error(err2)
		return []Project{}
	}

	images := retrieveAllImages()
	imagesById := map[string]Image{}
	for _, i := range images {
		imagesById[i.Id] = i
	}

	tests := retrieveAllTests()
	testsById := map[string]Test{}
	for _, t := range tests {
		testsById[t.Id] = t
	}

	for _, p := range projects {
		project := unmarshalProject(p)
		for _, i := range project.ImageIds {
			project.Images = append(project.Images, imagesById[i])
		}
		for _, t := range project.TestIds {
			project.Tests = append(project.Tests, testsById[t])
		}
		result = append(result, project)
	}

	return result
}

func retrieveAllImages() []Image {
	result := []Image{}
	rows, err := r.Table(tblNameImages).Run(session)
	if err != nil {
		log.Error(err)
		return []Image{}
	}
	images := []*model.Image{}
	err2 := rows.All(&images)
	if err2 != nil {
		log.Error(err2)
		return []Image{}
	}
	for _, p := range images {
		result = append(result, unmarshalImage(p))
	}
	return result
}

func retrieveAllResults() []BuildResult {
	result := []BuildResult {}
	rows, err := r.Table(tblNameBuilds).Run(session)
	if err != nil {
		log.Error(err)
		return []BuildResult{}
	}
	builds := []*model.Build{}
	err2 := rows.All(&builds)
	if err2 != nil {
		log.Error(err2)
		return []BuildResult{}
	}
	for _, p := range builds {
		for _, r := range p.Results {
			result = append(result, unmarshalResult(r))
		}
	}
	return result
}

func retrieveAllTests() []Test {
	result := []Test{}
	rows, err := r.Table(tblNameTests).Run(session)
	if err != nil {
		log.Error(err)
		return []Test{}

	}
	tests := []*model.Test{}
	err2 := rows.All(&tests)
	if err2 != nil {
		log.Error(err2)
		return []Test{}
	}
	for _, p := range tests {
		result = append(result, unmarshalTest(p))

	}
	return result
}

func retrieveAllBuilds() []Build {
	result := []Build{}
	rows, err := r.Table(tblNameBuilds).Run(session)
	if err != nil {
		log.Error(err)
		return []Build{}
	}
	builds := []*model.Build{}
	err2 := rows.All(&builds)
	if err2 != nil {
		log.Error(err2)
		return []Build{}
	}
	for _, p := range builds {
		result = append(result, unmarshalBuild(p))
	}
	return result
}

func retrieveAllRegistries() []Registry {
	result := []Registry{}
	rows, err := r.Table(tblNameRegistries).Run(session)
	if err != nil {
		log.Error(err)
		return []Registry{}
	}
	registries := []*Registry{}
	err2 := rows.All(&registries)
	if err2 != nil {
		log.Error(err2)
		return []Registry{}
	}
	for _, p := range registries {
		result = append(result, unmarshalRegistry(p))
	}
	return result
}

func unmarshalImage(v interface{}) Image {
	image := Image{}
	marshaledBytes := marshalObject(v)
	unmarshaledBytes := json.Unmarshal(marshaledBytes, &image)
	if unmarshaledBytes != nil {
		panic(unmarshaledBytes)
	}
	return image
}

func unmarshalProject(v interface{}) Project {
	project := Project{}
	marshaledBytes := marshalObject(v)
	unmarshaledBytes := json.Unmarshal(marshaledBytes, &project)
	if unmarshaledBytes != nil {
		panic(unmarshaledBytes)
	}
	return project
}

func unmarshalBuild(v interface{}) Build {
	build := Build{}
	marshaledBytes := marshalObject(v)
	unmarshaledBytes := json.Unmarshal(marshaledBytes, &build)
	if unmarshaledBytes != nil {
		panic(unmarshaledBytes)
	}
	return build
}

func unmarshalTest(v interface{}) Test {
	test := Test{}
	marshaledBytes := marshalObject(v)
	unmarshaledBytes := json.Unmarshal(marshaledBytes, &test)
	if unmarshaledBytes != nil {
		panic(unmarshaledBytes)
	}

	return test
}

func unmarshalResult(v interface{}) BuildResult {
	results := BuildResult{}
	marshaledBytes := marshalObject(v)
	unmarshaledBytes := json.Unmarshal(marshaledBytes, &results)
	if unmarshaledBytes != nil {
		panic(unmarshaledBytes)
	}
	return results
}

func unmarshalRegistry(v interface{}) Registry {
	results := Registry{}
	marshaledBytes := marshalObject(v)
	unmarshaledBytes := json.Unmarshal(marshaledBytes, &results)
	if unmarshaledBytes != nil {
		panic(unmarshaledBytes)
	}
	return results
}

func marshalObject(v interface{}) []byte {
	vBytes, _ := json.Marshal(v)
	return vBytes
}

func CreateStatistics() CollectedData {
	data := CollectedData{}
	data.MAC = retrieveMACAddress()
	data.Username = "admin"
	data.Images = retrieveAllImages()
	data.Projects = retrieveAllProjects()
	data.Builds = retrieveAllBuilds()
	data.Tests = retrieveAllTests()
	data.Results = retrieveAllResults()
	data.Registries = retrieveAllRegistries()
	return data
}

func GetData(addr string) *bytes.Buffer {
	initDatabase(addr)

	data := CreateStatistics()
	b := new(bytes.Buffer)

	json.NewEncoder(b).Encode(data)
	return b
}