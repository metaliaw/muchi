// Package hosting Owns the lifetime of an application instance independently of its trigger.
package hosting

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/config"
)

const startupTimeout = 30 * time.Second

type Instance struct {
	Operations application.Operations
	Settings   config.Config
	Close      func() error
}

// Loader Reuses successful startup and retries failures on the next invocation.
// Build must release partial resources before returning an error.
type Loader struct {
	Logger   *slog.Logger
	Build    func(context.Context) (Instance, error)
	mu       sync.Mutex
	instance *Instance
}

func (l *Loader) LoadInstance() (Instance, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.instance != nil {
		return *l.instance, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()
	instance, err := l.Build(ctx)
	if err != nil {
		if l.Logger != nil {
			l.Logger.Error("Application Startup Failed", "error", err)
		}
		return Instance{}, err
	}
	l.instance = &instance
	return instance, nil
}

func (l *Loader) CloseInstance() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.instance == nil {
		return nil
	}
	instance := l.instance
	l.instance = nil
	if instance.Close == nil {
		return nil
	}
	return instance.Close()
}
