package awx

import (
	"net/http"
)

type Client struct {
	JobTemplateService   *JobTemplateService
	InventoriesService   *InventoriesService
	HostService          *HostService
	JobService           *JobService
	OrganizationsService *OrganizationsService
	GroupService         *GroupService
}

func NewClient(baseURL string, username string, password string) (*Client, error) {

	tokenAuth := BasicAuth{
		Username: username,
		Password: password,
	}

	return newClient(baseURL, &tokenAuth)
}

func NewClientWithToken(baseURL string, token string) (*Client, error) {

	tokenAuth := TokenAuth{
		Token: token,
	}

	return newClient(baseURL, &tokenAuth)
}

func newClient(baseURL string, auth IAuth) (*Client, error) {
	requester := Requester{
		Base:   baseURL,
		Auth:   auth,
		Client: http.DefaultClient,
	}

	client := Client{
		JobTemplateService: &JobTemplateService{
			Requester: &requester,
		},
		InventoriesService: &InventoriesService{
			Requester: &requester,
		},
		GroupService: &GroupService{
			Requester: &requester,
		},
		HostService: &HostService{
			Requester: &requester,
		},
		JobService: &JobService{
			Requester: &requester,
		},
		OrganizationsService: &OrganizationsService{
			Requester: &requester,
		},
	}

	return &client, nil
}
