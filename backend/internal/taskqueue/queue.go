package taskqueue

import "github.com/cangrejometralleta/muchi-api/internal/model"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
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

// nextWake Separates two Wake-ups born in the same Second.
var nextWake atomic.Uint64

type Queue struct {
	client           *cloudtasks.Client
	parent           string
	targetURL        string
	serviceAccount   string
	dispatchDeadline time.Duration
}

// OpenQueue Connects Card Tasks through https://cloud.google.com/tasks/docs/reference/rest.
func OpenQueue(ctx context.Context, project, region, name, target, account string, deadline time.Duration) (*Queue, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("open task queue: %w", err)
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", project, region, name)
	return &Queue{client: client, parent: parent, targetURL: target, serviceAccount: account, dispatchDeadline: deadline}, nil
}

func (q *Queue) DispatchSearch(ctx context.Context, job model.Job) error {
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

// WakeWorker Adds one Turn that is not tied to any Position.
//
// The Sweeper uses it to replace Wake-ups that a silent Turn spent without
// working. The Name carries the Minute so a Repair is never mistaken for the
// original Task: Cloud Tasks remembers a used Name for about an Hour and
// answers AlreadyExists, which createTask swallows. Reusing the Name would
// repair nothing and report Success.
func (q *Queue) WakeWorker(ctx context.Context, reason string) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	return q.dispatch(ctx, fmt.Sprintf("%s/tasks/%s", q.parent, buildWakeName(reason)))
}

// buildWakeName Names one Wake-up so no two ever collide.
//
// Cloud Tasks admite Letras, Números, Guiones y Bajos: un Punto rompe la
// Llamada con InvalidArgument. Y un Sello por Segundos no alcanza, porque un
// Barrido pide muchos Despertares dentro del mismo Segundo: los repetidos
// volverían AlreadyExists, que dispatch se traga, y el Barredor repondría uno
// solo creyendo que repuso todos. El Sello se lee; el Contador separa.
func buildWakeName(reason string) string {
	stamp := time.Now().UTC().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%d", reason, stamp, nextWake.Add(1))
}

func (q *Queue) createTask(ctx context.Context, searchID string, position int) error {
	return q.dispatch(ctx, fmt.Sprintf("%s/tasks/%s-%03d", q.parent, searchID, position))
}

func (q *Queue) dispatch(ctx context.Context, name string) error {
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
		Name:             name,
		DispatchDeadline: durationpb.New(q.dispatchDeadline),
		MessageType:      &cloudtaskspb.Task_HttpRequest{HttpRequest: request},
	}
	_, err := q.client.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{Parent: q.parent, Task: task})
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	return err
}
