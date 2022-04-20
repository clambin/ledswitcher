package endpoint_test

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/endpoint"
	"github.com/clambin/ledswitcher/endpoint/led/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestEndpoints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ep1, ledSetter1, wg1 := startEndpoint(ctx, 8080)
	require.Eventually(t, func() bool { return ep1.IsRegistered() }, time.Second, 10*time.Millisecond)

	ep2, ledSetter2, wg2 := startEndpointWithLeaderPort(ctx, 8081, 8080)
	require.Eventually(t, func() bool { return ep2.IsRegistered() }, time.Second, 10*time.Millisecond)

	health := ep1.Broker.Stats()
	assert.Len(t, health.Endpoints, 2)

	health = ep2.Broker.Stats()
	assert.Len(t, health.Endpoints, 0)

	ledSetter1.On("SetLED", true).Return(nil).Once()
	ledSetter2.On("SetLED", true).Return(nil).Once()
	cancel()
	wg1.Wait()
	wg2.Wait()
	mock.AssertExpectationsForObjects(t, ledSetter1, ledSetter2)
}

func startEndpoint(ctx context.Context, port int) (ep *endpoint.Endpoint, ledSetter *mocks.Setter, wg *sync.WaitGroup) {
	return startEndpointWithLeaderPort(ctx, port, 0)
}

func startEndpointWithLeaderPort(ctx context.Context, port int, leaderPort int) (ep *endpoint.Endpoint, ledSetter *mocks.Setter, wg *sync.WaitGroup) {
	s, _ := scheduler.New("linear")
	b := broker.New(time.Second, s)
	ep = endpoint.New("127.0.0.1", port, "", b)
	ledSetter = &mocks.Setter{}
	ep.LEDSetter = ledSetter
	if leaderPort == 0 {
		ep.SetLeader("127.0.0.1")
	} else {
		ep.SetLeaderWithPort("127.0.0.1", leaderPort)
	}

	wg = &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err := ep.Run(ctx)
		if err != nil {
			panic("failed to start server: " + err.Error())
		}
		wg.Done()
	}()

	return
}

func doHTTPCall(url string, method string, body io.Reader) (statusCode int, err error) {
	req, _ := http.NewRequest(method, url, body)
	client := &http.Client{}
	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}
