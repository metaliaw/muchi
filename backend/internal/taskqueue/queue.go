package taskqueue

import (
	"context"
	"fmt"
	"sync"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maxRPCDeadline caps how far in the future a Cloud Tasks call's deadline can sit.
// The API rejects calls whose context deadline is more than ~30s out; an HTTP
// handler's context can carry a much longer one.
const maxRPCDeadline = 20 * time.Second

func boundContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= maxRPCDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, maxRPCDeadline)
}

type Queue struct {
	client         *cloudtasks.Client
	parent         string
	targetURL      string
	serviceAccount string
}

// OpenQueue Connects Card Tasks through https://cloud.google.com/tasks/docs/reference/rest.
func OpenQueue(ctx context.Context, project, region, name, target, account string) (*Queue, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("open task queue: %w", err)
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", project, region, name)
	return &Queue{client: client, parent: parent, targetURL: target, serviceAccount: account}, nil
}

func (q *Queue) DispatchSearch(ctx context.Context, job search.Job) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
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
