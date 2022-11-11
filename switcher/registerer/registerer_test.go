package registerer

import (
	"context"
	"errors"
	"github.com/clambin/ledswitcher/switcher/caller/mocks"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestRegisterer_Run(t *testing.T) {
	c := mocks.NewCaller(t)
	r := Registerer{
		Caller:      c,
		EndPointURL: "http://127.0.0.1:8080",
	}
	r.SetLeaderURL("http://127.0.0.1:8080")

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	go func() {
		r.Run(ctx)
	}()

	c.On("Register", "http://127.0.0.1:8080", "http://127.0.0.1:8080").Return(errors.New("fail")).Once()
	c.On("Register", "http://127.0.0.1:8080", "http://127.0.0.1:8080").Return(nil)

	require.Eventually(t, func() bool { return r.IsRegistered() }, 500*time.Millisecond, 100*time.Millisecond)

	cancel()
	wg.Wait()
}
