package api

import (
	"fmt"
	"github.com/gorilla/context"
	apiClient "github.com/shipyard/shipyard/client"
	"github.com/shipyard/shipyard/model"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"time"
)

const (
	PROJECT1_NAME      = "project 1"
	PROJECT2_NAME      = "project 2"
	PROJECT1_DESC      = "the first project"
	PROJECT2_DESC      = "the second project"
	PROJECT1_STATUS    = "new"
	PROJECT2_STATUS    = "bad status"
	PROJECT1_NAME2     = "project1 modified"
	PROJECT1_DESC2     = "the first project modified"
	PROJECT1_STATUS2   = "good status modified"
	PROJECT3_NAME      = "project 3"
	PROJECT3_DESC      = "the third project"
	PROJECT3_STATUS    = "no so good status"
	INVALID_USERNAME   = "batman"
	INVALID_PASSWORD   = "batmobile"
	INVALID_AUTH_TOKEN = "please let me in"
	IMAGE1_NAME        = "busybox"
	IMAGE1_DESC        = "my busybox image"
	IMAGE1_TAG         = "latest"
	IMAGE1_LOCATION    = "DockerHub Public Registry"
	IMAGE2_NAME        = "alpine"
	IMAGE2_DESC        = "my alpine image"
	IMAGE2_TAG         = "latest"
	IMAGE2_LOCATION    = "DockerHub Public Registry"
	IMAGE_ID           = "27end983nbd3bd8"
)

var (
	PROJECT1_SAVED_ID         string
	PROJECT1_PREV_UPDATE_TIME time.Time
	PROJECT3_SAVED_ID         string
	PROJECT_TO_RUN            string
)

func init() {
	dockerEndpoint := os.Getenv("SHIPYARD_DOCKER_URI")

	// Default docker endpoint
	if dockerEndpoint == "" {
		dockerEndpoint = "tcp://127.0.0.1:2375"
	}

	rethinkDbEndpoint := os.Getenv("SHIPYARD_RETHINKDB_URI")

	// Default rethinkdb endpoint
	if rethinkDbEndpoint == "" {
		rethinkDbEndpoint = "rethinkdb:28015"
	}

	localApi, localMux, err := InitServer(&ShipyardServerConfig{
		RethinkdbAddr:          rethinkDbEndpoint,
		RethinkdbAuthKey:       "",
		RethinkdbDatabase:      "shipyard_test",
		DisableUsageInfo:       true,
		ListenAddr:             "",
		AuthWhitelist:          []string{},
		EnableCors:             true,
		LdapServer:             "",
		LdapPort:               389,
		LdapBaseDn:             "",
		LdapAutocreateUsers:    true,
		LdapDefaultAccessLevel: "containers:ro",
		DockerUrl:              dockerEndpoint,
		TlsCaCert:              "",
		TlsCert:                "",
		TlsKey:                 "",
		AllowInsecure:          true,
		ShipyardTlsCert:        "",
		ShipyardTlsKey:         "",
		ShipyardTlsCACert:      "",
	})

	if err != nil {
		panic(fmt.Sprintf("Test init() for projects_test.go failed %s", err))
	}

	api = localApi
	globalMux = localMux

	cleanup()

	// Instantiate test server with Gorilla Mux Router enabled.
	// If you don't wrap the mux with the context.ClearHandler(),
	// then the server request cycle won't go through GorillaMux routing.
	ts = httptest.NewServer(context.ClearHandler(globalMux))

}

// TODO: this is snot cleaning up the tokens
func cleanup() error {

	if err := api.manager.DeleteAllProjects(); err != nil {
		return err
	}

	if err := api.manager.DeleteAllImages(); err != nil {
		return err
	}

	return nil
}

func TestProjectsGetAuthToken(t *testing.T) {

	Convey("Given a valid set of credentials", t, func() {
		Convey("When we make a successful request for an auth token", func() {
			header, err := apiClient.GetAuthToken(ts.URL, SYUSER, SYPASS)
			So(err, ShouldBeNil)

			Convey("Then we get a valid authentication header\n", func() {
				SY_AUTHTOKEN = header
				So(header, ShouldNotBeEmpty)
				numberOfParts := 2
				authToken := strings.SplitN(header, ":", numberOfParts)
				So(len(authToken), ShouldEqual, numberOfParts)
				So(authToken[0], ShouldEqual, SYUSER)
			})
		})

	})
}

func TestCreateNewProject(t *testing.T) {
	Convey("Given that we have a valid token", t, func() {
		So(SY_AUTHTOKEN, ShouldNotBeNil)
		So(SY_AUTHTOKEN, ShouldNotBeEmpty)
		Convey("When we make a request to create a new project", func() {
			id, code, err := apiClient.CreateProject(SY_AUTHTOKEN, ts.URL, PROJECT1_NAME, PROJECT1_DESC, PROJECT1_STATUS, nil, nil, nil, nil, false)
			Convey("Then we get back a successful response", func() {
				So(id, ShouldNotBeEmpty)
				So(err, ShouldBeNil)
				So(code, ShouldEqual, http.StatusCreated)
				PROJECT1_SAVED_ID = id
			})
		})
	})
}

//pull the project and make sure it is exactly as it was ordered to be created
func TestGetProject(t *testing.T) {
	Convey("Given that we have a valid project and a valid token", t, func() {
		So(SY_AUTHTOKEN, ShouldNotBeNil)
		So(SY_AUTHTOKEN, ShouldNotBeEmpty)
		So(PROJECT1_SAVED_ID, ShouldNotBeEmpty)

		Convey("When we make a request to retrieve it using its id", func() {
			project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID)
			Convey("Then the server should return OK", func() {
				So(err, ShouldBeNil)
				So(code, ShouldEqual, http.StatusOK)
				Convey("Then the returned project should have the expected values", func() {
					So(project.ID, ShouldEqual, PROJECT1_SAVED_ID)
					So(project.Description, ShouldEqual, PROJECT1_DESC)
					So(project.Name, ShouldEqual, PROJECT1_NAME)
					So(project.NeedsBuild, ShouldBeFalse)
					So(project.Status, ShouldEqual, PROJECT1_STATUS)
					PROJECT1_PREV_UPDATE_TIME = project.UpdateTime
				})
			})

		})
	})
}

func TestGetAllProjects(t *testing.T) {
	Convey("Given that we have created an additional project", t, func() {
		So(SY_AUTHTOKEN, ShouldNotBeNil)
		So(SY_AUTHTOKEN, ShouldNotBeEmpty)
		//So(PROJECT1_ID, ShouldNotBeEmpty)
		id, code, err := apiClient.CreateProject(SY_AUTHTOKEN, ts.URL, PROJECT2_NAME, PROJECT2_DESC, PROJECT2_STATUS, nil, nil, nil, nil, false)
		So(err, ShouldBeNil)
		So(id, ShouldNotBeEmpty)
		So(code, ShouldEqual, http.StatusCreated)
		Convey("When we make a request to retrieve all projects", func() {
			projects, code, err := apiClient.GetProjects(SY_AUTHTOKEN, ts.URL)
			Convey("Then the request should return some objects", func() {
				So(err, ShouldBeNil)
				So(code, ShouldEqual, http.StatusOK)
				So(projects, ShouldNotBeNil)
				So(len(projects), ShouldEqual, 2)
				Convey("And the objects return should have the expected structure and expected values.", func() {
					names := []string{}
					descriptions := []string{}
					statuses := []string{}
					ids := []string{}

					for _, project := range projects {
						names = append(names, project.Name)
						descriptions = append(descriptions, project.Description)
						statuses = append(statuses, project.Status)
						ids = append(ids, project.ID)
						So(project.ID, ShouldNotBeNil)
						So(project.ID, ShouldNotBeEmpty)
						So(project.LastRunTime, ShouldNotBeEmpty)
						So(project.UpdateTime, ShouldNotBeEmpty)
						So(project.CreationTime, ShouldNotBeEmpty)
						So(project.Author, ShouldNotBeEmpty)
					}

					So(PROJECT1_NAME, ShouldBeIn, names)
					So(PROJECT2_NAME, ShouldBeIn, names)
					So(PROJECT1_DESC, ShouldBeIn, descriptions)
					So(PROJECT2_DESC, ShouldBeIn, descriptions)
					So(PROJECT1_STATUS, ShouldBeIn, statuses)
				})
			})

		})

	})
}

func TestUpdateProject(t *testing.T) {
	Convey("Given that we have a project created already.", t, func() {
		Convey("When we request to update that project.", func() {
			code, err := apiClient.UpdateProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID, PROJECT1_NAME2, PROJECT1_DESC2,
				PROJECT1_STATUS2, nil, nil, nil, nil, true)
			Convey("Then we get an appropriate response back", func() {
				So(err, ShouldBeNil)
				So(code, ShouldEqual, http.StatusNoContent)
				Convey("And when we retrieve the project again, it has the modified values.", func() {
					project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID)
					So(err, ShouldBeNil)
					So(code, ShouldEqual, http.StatusOK)
					So(project.Name, ShouldEqual, PROJECT1_NAME2)
					So(project.Description, ShouldEqual, PROJECT1_DESC2)
					So(project.Status, ShouldEqual, PROJECT1_STATUS2)
					So(project.NeedsBuild, ShouldBeTrue)
					So(project.UpdateTime, ShouldHappenAfter, PROJECT1_PREV_UPDATE_TIME)
				})
			})
		})
	})
}

func TestDeleteProject(t *testing.T) {
	Convey("Given that we have a project created already.", t, func() {
		Convey("When we request to delete the project", func() {
			//delete the second project
			code, err := apiClient.DeleteProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID)
			Convey("Then we get confirmation that it was deleted.", func() {
				So(err, ShouldBeNil)
				So(code, ShouldEqual, http.StatusNoContent)
				Convey("And if we try to retrieve the project again by its id it should fail.", func() {
					//try to get the second project and make sure the server sends an error
					_, code, err = apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID)
					So(err, ShouldBeNil)
					So(code, ShouldEqual, http.StatusNotFound)
					Convey("And if we get all projects, it should not be in the collection.", func() {
						projects, code, err := apiClient.GetProjects(SY_AUTHTOKEN, ts.URL)
						So(err, ShouldBeNil)
						So(code, ShouldEqual, http.StatusOK)
						So(projects, ShouldNotBeNil)
						So(len(projects), ShouldEqual, 1)
						names := []string{}
						ids := []string{}

						for _, project := range projects {
							names = append(names, project.Name)
							ids = append(ids, project.ID)
							So(project.ID, ShouldNotBeNil)
							So(project.ID, ShouldNotBeEmpty)
							So(project.LastRunTime, ShouldNotBeEmpty)
							So(project.UpdateTime, ShouldNotBeEmpty)
							So(project.CreationTime, ShouldNotBeEmpty)
							So(project.Author, ShouldNotBeEmpty)
						}

						So(PROJECT1_SAVED_ID, ShouldNotBeIn, ids)

					})
				})
			})
		})
	})

}

func TestProjectNotFoundScenarios(t *testing.T) {
	cleanup()
	Convey("Given that a project with a given id does not exist", t, func() {
		Convey("When we try to retrieve that project by its id", func() {
			project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID)
			Convey("Then we should get a not found error", func() {
				So(project, ShouldBeNil)
				So(code, ShouldEqual, http.StatusNotFound)
				So(err, ShouldBeNil)
			})
		})
		Convey("When we try to delete that project by its id", func() {
			code, err := apiClient.DeleteProject(SY_AUTHTOKEN, ts.URL, PROJECT1_SAVED_ID)
			Convey("Then we should get a not found error", func() {
				So(code, ShouldEqual, http.StatusNotFound)
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestGetAuthTokenWithInvalidCredentials(t *testing.T) {
	Convey("Given that we have invalid credentials", t, func() {
		Convey("When we try to request an auth token", func() {
			token, err := apiClient.GetAuthToken(ts.URL, INVALID_USERNAME, INVALID_PASSWORD)
			Convey("Then we should get an error", func() {
				So(err, ShouldNotBeNil)
				Convey("And response should not contain any token", func() {
					So(token, ShouldBeBlank)
				})
			})
		})
	})
}

func TestUnauthorizedProjectRequests(t *testing.T) {
	Convey("Given that we don't have a valid token", t, func() {
		Convey("When we try to get all projects", func() {
			projects, code, err := apiClient.GetProjects(INVALID_AUTH_TOKEN, ts.URL)
			Convey("Then we should be denied access", func() {
				So(code, ShouldEqual, http.StatusUnauthorized)
				So(err, ShouldNotBeNil)
				Convey("And we should not get anything in return", func() {
					So(projects, ShouldBeNil)
				})
			})
		})
	})
	Convey("Given that we have an empty token", t, func() {
		Convey("When we request to create a new project", func() {
			id, code, err := apiClient.CreateProject("", ts.URL, PROJECT3_NAME, PROJECT3_DESC, PROJECT3_STATUS, nil, nil, nil, nil, false)
			Convey("Then we should be denied access", func() {
				So(err, ShouldNotBeNil)
				So(code, ShouldEqual, http.StatusUnauthorized)
				Convey("And we shoudl not get anything in return", func() {
					So(id, ShouldBeBlank)
				})
			})
		})
	})
}

func TestAddProjectImage(t *testing.T) {
	Convey("Given that we create a new project", t, func() {
		projectId, code, err := apiClient.CreateProject(SY_AUTHTOKEN, ts.URL, PROJECT3_NAME, PROJECT3_DESC, PROJECT3_STATUS, nil, nil, nil, nil, false)
		So(projectId, ShouldNotBeEmpty)
		So(code, ShouldEqual, http.StatusCreated)
		So(err, ShouldBeNil)
		Convey("When we add a new image to it", func() {
			code, err := apiClient.AddProjectImage(
				SY_AUTHTOKEN,
				ts.URL,
				projectId,
				IMAGE1_NAME,
				IMAGE_ID,
				IMAGE1_TAG,
				[]string{"latest", "awesomeTag"},
				IMAGE1_DESC,
				// TODO: it would be best to actually create a new registry object first and use that id.
				"myregistryId123423412342",
				IMAGE1_LOCATION,
				true,
			)
			Convey("Then we should get a successful response", func() {
				So(code, ShouldEqual, http.StatusCreated)
				So(err, ShouldBeNil)
				Convey("And the project should now have an image embedded in its structure", func() {
					project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, projectId)
					So(err, ShouldBeNil)
					So(project, ShouldNotBeNil)
					So(code, ShouldEqual, http.StatusOK)
					imageNames := []string{}
					So(len(project.Images), ShouldEqual, 1)
					for _, image := range project.Images {
						imageNames = append(imageNames, image.Name)
					}
					So(IMAGE1_NAME, ShouldBeIn, imageNames)
					PROJECT3_SAVED_ID = projectId
				})
			})
		})

	})
}

func TestAddProjectTest(t *testing.T) {
	Convey("When we add a new test to it", t, func() {
		id, code, err := apiClient.CreateTest(
			SY_AUTHTOKEN,
			ts.URL,
			"testName",
			"description",
			[]*model.TargetArtifact{
				&model.TargetArtifact{
					ID:           IMAGE_ID,
					ArtifactType: "image",
				},
			},
			"",
			"Clair [Internal]",
			"",
			"",
			PROJECT3_SAVED_ID,
			nil,
			"s",
			"f",
			"",
		)
		Convey("Then we should get a successful response", func() {
			So(code, ShouldEqual, http.StatusCreated)
			So(id, ShouldNotBeNil)
			So(err, ShouldBeNil)
			Convey("And the project should now have an image embedded in its structure", func() {
				project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID)
				So(err, ShouldBeNil)
				So(project, ShouldNotBeNil)
				So(code, ShouldEqual, http.StatusOK)
				So(len(project.Tests), ShouldEqual, 1)
			})

			PROJECT_TO_RUN = PROJECT3_SAVED_ID
		})
	})
}

func TestExecuteProject(t *testing.T) {

	Convey("When we make a request to create a new project", t, func() {
		id, code, err := apiClient.CreateProject(SY_AUTHTOKEN, ts.URL, PROJECT1_NAME, PROJECT1_DESC, PROJECT1_STATUS, nil, nil, nil, nil, false)
		Convey("Then we get back a successful response", func() {
			So(id, ShouldNotBeEmpty)
			So(err, ShouldBeNil)
			So(code, ShouldEqual, http.StatusCreated)
			PROJECT_TO_RUN = id
		})
	})

	Convey("When we make a request to create a new image", t, func() {
		id, err := apiClient.CreateImage(SY_AUTHTOKEN, ts.URL, PROJECT_TO_RUN, "alpine", "id1", "latest", []string{}, "description", "public registry", "public registry", true)
		Convey("Then we get back a successful response", func() {
			So(err, ShouldBeNil)
			So(id, ShouldNotBeEmpty)
			IMAGE_SAVED_ID = id
		})
	})

	Convey("When we make a request to create a new test", t, func() {

		id, code, err := apiClient.CreateTest(SY_AUTHTOKEN, ts.URL, TEST1_NAME, TEST1_DESC, TEST_ARTIFACTS, TEST1_TYPE, "provider type", "provider name", "provider test", PROJECT_TO_RUN, []*model.Parameter{}, "success tag", "fail tag", "from tag")

		Convey("Then we get back a successful response", func() {
			So(err, ShouldBeNil)
			So(code, ShouldEqual, http.StatusCreated)
			So(id, ShouldNotBeEmpty)
			TEST_ID = id
		})
	})

	Convey("Given that we execute the project we just created", t, func() {
		status, err := apiClient.RunProject(SY_AUTHTOKEN, ts.URL, PROJECT_TO_RUN)
		So(status, ShouldEqual, http.StatusCreated)
		So(err, ShouldBeNil)

		Convey("Then we should get a successful response", func() {
			var project *model.Project

			timeout := true
			go func() {
				time.Sleep(time.Second * 60)
				timeout = false
			}()
			for timeout {
				p, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT_TO_RUN)
				So(err, ShouldBeNil)
				So(p, ShouldNotBeNil)
				So(code, ShouldEqual, http.StatusOK)

				if p.ActionStatus == model.ProjectFinishedActionLabel {
					project = p
					break
				}
				time.Sleep(time.Second * 5)
			}

			So(project.ActionStatus, ShouldEqual, model.ProjectFinishedActionLabel)
			So(project.Status, ShouldEqual, model.BuildStatusFinishedSuccess)
		})
	})
}

func TestAddProjectImageViaUpdate(t *testing.T) {
	Convey("Given that we have a previous project with an image", t, func() {
		So(PROJECT3_SAVED_ID, ShouldNotBeBlank)
		project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID)
		So(err, ShouldBeNil)
		So(project, ShouldNotBeNil)
		So(code, ShouldEqual, http.StatusOK)
		imageIdsList := project.ImageIds
		imagesFromServer := []*model.Image{}

		Convey("When we make a request to add another image to the project", func() {
			id, err := apiClient.CreateImage(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID, "alpine", "id1", "latest", []string{}, "description", "public registry", "public registry", true)
			Convey("Then we get back a successful response", func() {
				So(err, ShouldBeNil)
				So(id, ShouldNotBeEmpty)
				imageIdsList = append(imageIdsList, id)
			})
		})

		Convey("When we request an update of the project", func() {
			code, err := apiClient.UpdateProject(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID, project.Name, project.Description, project.Status, nil, project.Images, nil, nil, true)
			Convey("Then we should get a successful response", func() {
				So(err, ShouldBeNil)
				So(code, ShouldEqual, http.StatusNoContent)
				Convey("And we should be able to get the project with all the images that it should have", func() {
					project, code, err := apiClient.GetProject(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID)
					So(err, ShouldBeNil)
					So(project, ShouldNotBeNil)
					So(code, ShouldEqual, http.StatusOK)
					So(imageIdsList, ShouldResemble, project.ImageIds)
					So(len(project.ImageIds), ShouldEqual, 2)
				})
			})
		})

		Convey("Given that we have added two images to the previously created project", func() {
			So(PROJECT3_SAVED_ID, ShouldNotBeBlank)
			Convey("When we request the same images using a nested projects route", func() {
				images, code, err := apiClient.GetProjectImages(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID)
				So(code, ShouldEqual, http.StatusOK)
				So(err, ShouldBeNil)
				Convey("Then we should get the same images as before", func() {
					names := []string{}
					So(len(images), ShouldEqual, 2)
					for _, image := range images {
						names = append(names, image.Name)
						So(image.ID, ShouldNotBeBlank)
						imagesFromServer = append(imagesFromServer, image)
					}

					So("busybox", ShouldBeIn, names)
					So("alpine", ShouldBeIn, names)
					Convey("And we should be able to retrieve those images individually through the nested route.", func() {
						for _, currentImage := range imagesFromServer {
							image, code, err := apiClient.GetProjectImage(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID, currentImage.ID)
							So(image, ShouldNotBeNil)
							So(code, ShouldEqual, http.StatusOK)
							So(err, ShouldBeNil)
							So(image.ID, ShouldEqual, currentImage.ID)
						}
						Convey("And we should be able to remove those images by id using the nested route", func() {
							for _, currentImage := range imagesFromServer {
								code, err := apiClient.DeleteProjectImage(SY_AUTHTOKEN, ts.URL, PROJECT3_SAVED_ID, currentImage.ID)
								So(code, ShouldEqual, http.StatusNoContent)
								So(err, ShouldBeNil)
							}
						})
					})
				})

			})
		})
	})

}

// This is a hack to ensure teardown / cleanup after this test suite ends.
// TODO: Add functionality for manager to close the database session
func TestCleanupProjectTests(t *testing.T) {
	// Cleanup all the state in the database
	Convey("Given that we have finished our project test suite", t, func() {
		Convey("Then we can cleanup", func() {
			err := cleanup()
			So(err, ShouldBeNil)
			//Convey("And we should be able to shutdown the server", func() {
			//	ts.Close()
			//})
		})
	})

}
