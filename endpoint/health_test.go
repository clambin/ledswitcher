package endpoint_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestEndpoint_Health(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ep, ledSetter, wg := startEndpoint(ctx, 0)
	ledSetter.On("SetLED", mock.AnythingOfType("bool")).Return(nil)

	require.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", ep.HTTPServer.Port))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel()
	wg.Wait()
}
