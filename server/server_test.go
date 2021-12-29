package server_test

import (
	"context"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/endpoint/led/mocks"
	"github.com/clambin/ledswitcher/server"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	s := server.New("127.0.0.1", 0, "", 10*time.Millisecond, &scheduler.LinearScheduler{}, "127.0.0.1")
	ledSetter := &mocks.Setter{}
	s.Endpoint.LEDSetter = ledSetter

	ctx, cancel := context.WithCancel(context.Background())

	ledSetter.On("SetLED", true).Return(nil)
	ledSetter.On("SetLED", false).Return(nil)

	s.Start(ctx)

	require.Eventually(t, func() bool { return s.Endpoint.IsRegistered() }, time.Second, 10*time.Millisecond)

	time.Sleep(200 * time.Millisecond)

	cancel()
	s.Wait()

	mock.AssertExpectationsForObjects(t, ledSetter)
}
