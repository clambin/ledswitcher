package driver_test

import (
	"context"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/leader/driver"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestDriver_Run(t *testing.T) {
	l, _ := driver.New(configuration.LeaderConfiguration{
		Rotation:  100 * time.Millisecond,
		Scheduler: configuration.SchedulerConfiguration{Mode: "linear"},
	}, http.DefaultClient, slog.Default().With("component", "leader"))

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error)
	go func() {
		ch <- l.Run(ctx)
	}()

	l.SetLeading(true)

	l.RegisterClient("http://foo:1234")
	l.RegisterClient("http://bar:1234")

	time.Sleep(500 * time.Millisecond)

	cancel()
	<-ch

	stats := l.Scheduler.GetHosts()
	assert.Len(t, stats, 2)
}

func TestDriver_Fail(t *testing.T) {
	_, err := driver.New(configuration.LeaderConfiguration{
		Scheduler: configuration.SchedulerConfiguration{Mode: "<invalid>"},
	}, http.DefaultClient, slog.Default())
	assert.Error(t, err)
}
