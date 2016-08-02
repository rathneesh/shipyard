package ilm_client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"github.com/shipyard/shipyard/model"
	"github.com/shipyard/shipyard/model/dockerhub"
	"strings"
)

func DockerHubSearchImage(authHeader, url string, imageName string) ([]dockerhub.Image, int, error) {
	resp, err := sendRequest(authHeader, "GET", fmt.Sprintf("%s/api/v1/search?q=%s", url, imageName), "")
	if err != nil {
		return nil, resp.StatusCode, err
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, err
		}
		type results struct {
			Results []dockerhub.Image `json:"results"`
		}
		var r results
		err = json.Unmarshal(body, &r)
		if err != nil {
			return nil, resp.StatusCode, err
		} else {
			return r.Results, resp.StatusCode, nil
		}
	}
}
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
func DockerHubSearchImageTags(authHeader, url string, imageName string) ([]string, int, error) {
	 if !strings.Contains(imageName,"/"){
		 imageName = "library/" + imageName
	}
	token := getDockerHubToken(imageName)
	header := "Bearer "+ token
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://registry-1.docker.io/v2/" + imageName + "/tags/list", nil)
	req.Header.Set("Authorization", header)
	resp, err := client.Do(req)
	if err != nil {
		return nil, resp.StatusCode, err
	} else {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, resp.StatusCode, err
		}
		var tags dockerhub.TagV2
		err = json.Unmarshal(body, &tags)
		if err != nil {
			return nil, resp.StatusCode, err
		} else {
			return tags.Tags, resp.StatusCode, nil
		}
	}
}
