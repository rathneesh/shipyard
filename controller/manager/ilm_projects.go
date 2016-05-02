package manager

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/shipyard/shipyard/model"
	"time"
)

// methods related to the Project structure
func (m DefaultManager) Projects() ([]*model.Project, error) {
	// TODO: consider making sorting customizable
	// TODO: should filter by authorization
	// Return all projects **WITHOUT** their images embedded
	res, err := r.Table(tblNameProjects).OrderBy(r.Asc("creationTime")).Run(m.session)
	if err != nil {
		return nil, err
	}
	projects := []*model.Project{}
	if err := res.All(&projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func (m DefaultManager) Project(id string) (*model.Project, error) {

	var project *model.Project

	res, err := r.Table(tblNameProjects).Filter(map[string]string{"id": id}).Run(m.session)

	if err != nil {
		return nil, err
	}
	if res.IsNil() {
		return nil, ErrProjectDoesNotExist
	}
	if err := res.One(&project); err != nil {
		return nil, err
	}

	project.Images, err = m.GetImages(project.ID)

	if err != nil {
		return nil, ErrProjectImagesProblem
	}
	project.Tests, err = m.GetTests(project.ID)

	if err != nil {
		return nil, ErrProjectTestsProblem
	}

	return project, nil
}

func (m DefaultManager) SaveProject(project *model.Project) error {
	var eventType string
	proj, err := m.Project(project.ID)

	if err != nil && err != ErrProjectDoesNotExist {
		return err
	}
	if proj != nil {
		return ErrProjectExists
	}
	project.CreationTime = time.Now().UTC()
	project.UpdateTime = project.CreationTime
	// TODO: find a way to retrieve the current user
	project.Author = "author"

	//create the project
	response, err := r.Table(tblNameProjects).Insert(project).RunWrite(m.session)

	if err != nil {
		return err
	}

	// rethinkDB returns the ID as the first element of the GeneratedKeys slice
	// TODO: this method seems brittle, should contact the gorethink dev team for insight on this.
	project.ID = func() string {
		if len(response.GeneratedKeys) > 0 {
			return string(response.GeneratedKeys[0])
		}
		return ""
	}()

	//add the project ID to the images and save them in the Images table
	// TODO: investigate how to do a bulk insert
	for _, img := range project.Images {
		img.ProjectId = project.ID
		response, err = r.Table(tblNameImages).Insert(img).RunWrite(m.session)

		if err != nil {
			return err
		}
	}
	//add project ID to the tests and save them in the Tests table
	// TODO: investigate how to do a bulk insert
	for _, test := range project.Tests {
		test.ProjectId = project.ID
		response, err = r.Table(tblNameTests).Insert(test).RunWrite(m.session)

		if err != nil {
			return err
		}
	}

	eventType = "add-project"
	m.logEvent(eventType, fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})
	return nil
}

func (m DefaultManager) UpdateProject(project *model.Project) error {
	var eventType string
	// check if exists; if so, update
	proj, err := m.Project(project.ID)
	if err != nil && err != ErrProjectDoesNotExist {
		return err
	}
	// update
	if proj != nil {
		updates := map[string]interface{}{
			"name":        project.Name,
			"description": project.Description,
			"status":      project.Status,
			"needsBuild":  project.NeedsBuild,
			"updateTime":  time.Now().UTC(),
			// TODO: find a way to retrieve the current user
			"updatedBy": "updater",
		}

		//TODO: Find a more elegant approach
		// Retrieve images by projectId and delete them by their primary key id generated by rethink
		res, err := r.Table(tblNameImages).Filter(map[string]string{"projectId": proj.ID}).Run(m.session)
		if err != nil {
			return err
		}
		oldImages := []*model.Image{}
		if err := res.All(&oldImages); err != nil {
			return err
		}

		// Remove existing images for this project
		for _, oldImage := range oldImages {
			if _, err := r.Table(tblNameImages).Filter(map[string]string{"id": oldImage.ID}).Delete().Run(m.session); err != nil {
				return err
			}
		}

		// Insert all the images that are incoming from the request which should have the new and old ones
		// TODO: investigate how we can do bulk insert
		for _, newImage := range project.Images {
			newImage.ProjectId = proj.ID
			if _, err := r.Table(tblNameImages).Insert(newImage).RunWrite(m.session); err != nil {
				return err
			}
		}

		//TODO: Find a more elegant approach
		// Retrieve tests by projectId and delete them by their primary key id generated by rethink
		res, err = r.Table(tblNameTests).Filter(map[string]string{"projectId": proj.ID}).Run(m.session)
		if err != nil {
			return err
		}
		oldTests := []*model.Test{}
		if err := res.All(&oldTests); err != nil {
			return err
		}

		// Remove existing tests for this project
		for _, oldTest := range oldTests {
			if _, err := r.Table(tblNameTests).Filter(map[string]string{"id": oldTest.ID}).Delete().Run(m.session); err != nil {
				return err
			}
		}

		// Insert all the tests that are incoming from the request which should have the new and old ones
		// TODO: investigate how we can do bulk insert
		for _, newTest := range project.Tests {
			newTest.ProjectId = proj.ID
			if _, err := r.Table(tblNameTests).Insert(newTest).RunWrite(m.session); err != nil {
				return err
			}
		}
		if _, err := r.Table(tblNameProjects).Filter(map[string]string{"id": project.ID}).Update(updates).RunWrite(m.session); err != nil {
			return err
		}

		eventType = "update-project"
	}

	m.logEvent(eventType, fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})

	return nil
}

func (m DefaultManager) DeleteProject(project *model.Project) error {
	res, err := r.Table(tblNameProjects).Filter(map[string]string{"id": project.ID}).Delete().Run(m.session)
	if err != nil {
		return err
	}

	if res.IsNil() {
		return ErrProjectDoesNotExist
	}
	res, err = r.Table(tblNameImages).Filter(map[string]string{"projectId": project.ID}).Run(m.session)
	if err != nil {
		return err
	}
	imagesToDelete := []*model.Image{}
	if err := res.All(&imagesToDelete); err != nil {
		return err
	}

	// Remove existing images for this project
	for _, imgToDelete := range imagesToDelete {
		if _, err := r.Table(tblNameImages).Filter(map[string]string{"id": imgToDelete.ID}).Delete().Run(m.session); err != nil {
			return err
		}
	}

	res, err = r.Table(tblNameTests).Filter(map[string]string{"projectId": project.ID}).Run(m.session)
	if err != nil {
		return err
	}
	testsToDelete := []*model.Test{}
	if err := res.All(&testsToDelete); err != nil {
		return err
	}

	// Remove existing tests for this project
	for _, testToDelete := range testsToDelete {
		if _, err := r.Table(tblNameTests).Filter(map[string]string{"id": testToDelete.ID}).Delete().Run(m.session); err != nil {
			return err
		}
	}

	m.logEvent("delete-project", fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})

	return nil
}

func (m DefaultManager) DeleteAllProjects() error {
	_, err := r.Table(tblNameProjects).Delete().Run(m.session)

	if err != nil {
		return err
	}

	return nil
}

// end methods related to the project structure
