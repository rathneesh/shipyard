package api

import (
	"io/ioutil"
	"net/http"
	"encoding/json"
	"github.com/shipyard/shipyard/model"
	"strings"
)
// code to get all tags from dockerhub


func getDockerHubToken(imageName string ) string {
	jsonToken := model.DockerHubV2Token{}
	response, err := http.Get("https://auth.docker.io/token?service=registry.docker.io&scope=repository:" + imageName + ":pull")
	if err != nil {
		return string(err.Error())
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return string(err.Error())
	}
	err = json.Unmarshal(contents, &jsonToken)
	if err != nil {
		return string(err.Error())
	}
	return jsonToken.Token
}
// dockerhub forward POC
func (a *Api) dockerhubSearch(w http.ResponseWriter, r *http.Request) {
	// TODO: make an actual proxy using the `github.com/mailgun/oxy/forward` package (note: client cannot change host during forwarding)
	w.Header().Set("content-type", "application/json")

	query := r.URL.Query().Get("q")
	if !strings.Contains(query,"/"){
		query = "library/" + query
	}
	response, err := http.Get("https://registry.hub.docker.com/v2/repositories/" + query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write(contents)
}

// get the tags of an image from dockerhub
func (a *Api) dockerhubTags(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	repo := r.URL.Query().Get("r")
	if !strings.Contains(repo,"/"){
		repo = "library/" + repo
	}
	token := getDockerHubToken(repo)
	header := "Bearer "+ token
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://registry-1.docker.io/v2/" + repo + "/tags/list", nil)
	req.Header.Set("Authorization", header)
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), res.StatusCode)
		return
	}
	defer res.Body.Close()

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		http.Error(w, err.Error(), res.StatusCode)
		return
	}
	w.Write(contents)
}
