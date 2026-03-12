package transcoder

import (
	"context"
	"fmt"

	transcoder "cloud.google.com/go/video/transcoder/apiv1"
	transcoderpb "cloud.google.com/go/video/transcoder/apiv1/transcoderpb"
)

type Client struct {
	client     *transcoder.Client
	projectID  string
	location   string
	templateID string
}

func NewClient(ctx context.Context, projectID, location, templateID string) (*Client, error) {
	client, err := transcoder.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transcoder client: %w", err)
	}

	return &Client{
		client:     client,
		projectID:  projectID,
		location:   location,
		templateID: templateID,
	}, nil
}

func (c *Client) CreateJob(ctx context.Context, inputURI, outputURI string) (string, error) {
	req := &transcoderpb.CreateJobRequest{
		Parent: c.getParent(),
		Job: &transcoderpb.Job{
			InputUri:  inputURI,
			OutputUri: outputURI,
			JobConfig: &transcoderpb.Job_TemplateId{
				TemplateId: "hls-adaptive-template",
			},
		},
	}

	job, err := c.client.CreateJob(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create transcoder job: %w", err)
	}

	return job.Name, nil
}

func (c *Client) GetJob(ctx context.Context, jobName string) (*transcoderpb.Job, error) {
	req := &transcoderpb.GetJobRequest{
		Name: jobName,
	}

	job, err := c.client.GetJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get transcoder job: %w", err)
	}

	return job, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) getParent() string {
	return fmt.Sprintf("projects/%s/locations/%s", c.projectID, c.location)
}
