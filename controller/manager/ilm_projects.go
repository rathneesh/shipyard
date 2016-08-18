package manager

import (
	"fmt"
	r "github.com/dancannon/gorethink"
	"github.com/shipyard/shipyard/model"
	"time"
	//"github.com/shipyard/shipyard/controller/manager/longpoll"
)

// methods related to the Project structure
func (m DefaultManager) Projects() ([]*model.Project, error) {
	// TODO: consider making sorting customizable
	// TODO: should filter by authorization
	// Return all projects **WITHOUT** their images embedded
	res, err := r.Table(tblNameProjects).OrderBy(r.Asc("creationTime")).Run(m.session)
	defer res.Close()
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
	var images []*model.Image
	var tests []*model.Test

	res, err := r.Table(tblNameProjects).Filter(map[string]string{"id": id}).Run(m.session)
	defer res.Close()
	if err != nil {
		return nil, err
	}
	if res.IsNil() {
		return nil, ErrProjectDoesNotExist
	}
	if err := res.One(&project); err != nil {
		return nil, err
	}
	for _, imgid := range project.ImageIds {
		img, _ := m.GetImage(imgid)
		images = append(images, img)
	}
	project.Images = append(project.Images, images...)

	for _, testid := range project.TestIds {
		test, _ := m.GetTest(testid)
		tests = append(tests, test)
	}
	project.Tests = append(project.Tests, tests...)

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
	project.ActionStatus = model.ProjectNewActionLabel
	project.Status = model.BuildStatusNewLabel
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
		response, err = r.Table(tblNameImages).Insert(img).RunWrite(m.session)

		if err != nil {
			return err
		}
		img.ID = func() string {
			if len(response.GeneratedKeys) > 0 {
				return string(response.GeneratedKeys[0])
			}
			return ""
		}()
		project.ImageIds = append(project.ImageIds, img.ID)

	}
	//add project ID to the tests and save them in the Tests table
	// TODO: investigate how to do a bulk insert
	for _, test := range project.Tests {
		response, err = r.Table(tblNameTests).Insert(test).RunWrite(m.session)

		if err != nil {
			return err
		}

		test.ID = func() string {
			if len(response.GeneratedKeys) > 0 {
				return string(response.GeneratedKeys[0])
			}
			return ""
		}()
		project.TestIds = append(project.TestIds, test.ID)
	}

	eventType = "add-project"
	// after we add the images/tests and generate their ids we insert them into the project's imagedids/testids list
	// and then we update it
	m.UpdateProject(project)
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
		proj.Name = project.Name
		proj.Description = project.Description
		proj.Status = project.Status
		proj.ActionStatus = project.ActionStatus
		proj.NeedsBuild = project.NeedsBuild
		proj.UpdatedBy = "updater"
		proj.UpdateTime = time.Now().UTC()
		if _, err := r.Table(tblNameProjects).Filter(map[string]string{"id": proj.ID}).Update(proj).RunWrite(m.session); err != nil {
			return err
		}

		eventType = "update-project"
	}
	m.logEvent(eventType, fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})

	return nil
}

func (m DefaultManager) UpdateTestIds(project *model.Project) error {
	var eventType string
	proj, err := m.Project(project.ID)
	if err != nil && err != ErrProjectDoesNotExist {
		return err
	}
	if proj != nil {
		proj.TestIds = project.TestIds
	}

	if _, err := r.Table(tblNameProjects).Filter(map[string]string{"id": proj.ID}).Update(proj).RunWrite(m.session); err != nil {
		return err

	}
	eventType = "update-project"
	m.logEvent(eventType, fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})

	return nil
}

func (m DefaultManager) UpdateImageIds(project *model.Project) error {
	var eventType string
	proj, err := m.Project(project.ID)
	if err != nil && err != ErrProjectDoesNotExist {
		return err
	}
	if proj != nil {
		proj.ImageIds = project.ImageIds
	}

	if _, err := r.Table(tblNameProjects).Filter(map[string]string{"id": proj.ID}).Update(proj).RunWrite(m.session); err != nil {
		return err
	}

	eventType = "update-project"
	m.logEvent(eventType, fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})

	return nil
}
func (m DefaultManager) DeleteProject(project *model.Project) error {
	res, err := r.Table(tblNameProjects).Filter(map[string]string{"id": project.ID}).Delete().Run(m.session)
	defer res.Close()
	if err != nil {
		return err
	}
	if res.IsNil() {
		return ErrProjectDoesNotExist
	}

	// Remove existing images for this project
	for _, imgIdToDelete := range project.ImageIds {
		res, err := r.Table(tblNameImages).Filter(map[string]string{"id": imgIdToDelete}).Delete().Run(m.session)
		if err != nil {
			return err
		}
		res.Close()
	}

	// Remove existing tests for this project
	for _, testIdToDelete := range project.TestIds {
		res, err := r.Table(tblNameTests).Filter(map[string]string{"id": testIdToDelete}).Delete().Run(m.session)
		if err != nil {
			return err
		}
		res.Close()
	}

	m.logEvent("delete-project", fmt.Sprintf("id=%s, name=%s", project.ID, project.Name), []string{"security"})

	return nil
}

func (m DefaultManager) DeleteAllProjects() error {
	res, err := r.Table(tblNameProjects).Delete().Run(m.session)
	defer res.Close()
	if err != nil {
		return err
	}

	return nil
}
