package endpoint_test

import (
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/broker"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestEndpoint_Stats(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithCancel(context.Background())
	ep, ledSetter, wg := startEndpoint(ctx, 0)

	require.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	resp, err := http.Get(ep.MakeURL("127.0.0.1") + "/stats")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() {
		_ = resp.Body.Close()
	}()

	var body []byte
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	var stats broker.Stats
	err = json.Unmarshal(body, &stats)
	require.NoError(t, err)

	assert.True(t, stats.Leader)
	require.Len(t, stats.Endpoints, 1)
	assert.Equal(t, ep.MakeURL("127.0.0.1"), stats.Endpoints[0].Name)

	ledSetter.On("SetLED", true).Return(nil)
	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, ledSetter)
}
