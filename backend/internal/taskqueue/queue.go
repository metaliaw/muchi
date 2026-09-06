package taskqueue

import (
	"context"
	"fmt"
	"sync"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Queue struct {
	client         *cloudtasks.Client
	parent         string
	targetURL      string
	serviceAccount string
}

// OpenQueue Connects Card Tasks through https://cloud.google.com/tasks/docs/reference/rest.
func OpenQueue(ctx context.Context, project, region, name, target, account string) (*Queue, error) {
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("open task queue: %w", err)
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", project, region, name)
	return &Queue{client: client, parent: parent, targetURL: target, serviceAccount: account}, nil
}

func (q *Queue) DispatchSearch(ctx context.Context, job search.Job) error {
	errors := make(chan error, job.Total)
	var group sync.WaitGroup
	for position := range job.Total {
		group.Add(1)
		go func() {
			defer group.Done()
			errors <- q.createTask(ctx, job.ID, position)
		}()
	}
	group.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

func (q *Queue) CloseQueue() error {
	return q.client.Close()
}

func (q *Queue) createTask(ctx context.Context, searchID string, position int) error {
	name := fmt.Sprintf("%s/tasks/%s-%03d", q.parent, searchID, position)
	request := &cloudtaskspb.HttpRequest{
		Url:        q.targetURL,
		HttpMethod: cloudtaskspb.HttpMethod_POST,
		AuthorizationHeader: &cloudtaskspb.HttpRequest_OidcToken{
			OidcToken: &cloudtaskspb.OidcToken{
				ServiceAccountEmail: q.serviceAccount,
				Audience:            q.targetURL,
			},
		},
	}
	task := &cloudtaskspb.Task{
		Name:        name,
		MessageType: &cloudtaskspb.Task_HttpRequest{HttpRequest: request},
	}
	_, err := q.client.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{Parent: q.parent, Task: task})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}
