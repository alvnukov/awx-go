package playbook

import "github.com/alvnukov/awx-go"

type Playbook struct {
	client           *awx.Client
	organizationName string
}

func NewPlaybook(url string, token string, organization string) (*Playbook, error) {
	client, err := awx.NewClientWithToken(url, token)
	if err != nil {
		return nil, err
	}

	return &Playbook{
		client:           client,
		organizationName: organization,
	}, nil
}
