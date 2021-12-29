package endpoint_test

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestEndpoint_Register(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ep, ledSetter, wg := startEndpoint(ctx, 0)

	require.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	statusCode, err := doHTTPCall(
		ep.MakeURL("127.0.0.1")+"/register",
		http.MethodPost,
		bytes.NewBufferString(`{"url": "http://127.0.0.1:8888"}`),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, statusCode)

	statusCode, err = doHTTPCall(
		ep.MakeURL("127.0.0.1")+"/register",
		http.MethodPost,
		bytes.NewBufferString(`{"url": "http://127.0.0.1:8888"`),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, statusCode)

	ledSetter.On("SetLED", true).Return(nil).Once()

	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, ledSetter)
}

func TestEndpoint_Register_NotLeading(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ep, ledSetter, wg := startEndpoint(ctx, 0)

	require.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	ep.SetLeader("http://127.0.0.1:8888")

	statusCode, err := doHTTPCall(
		ep.MakeURL("127.0.0.1")+"/register",
		http.MethodPost,
		bytes.NewBufferString(`{"url": "http://127.0.0.1:8888"}`),
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, statusCode)

	ledSetter.On("SetLED", true).Return(nil).Once()
	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, ledSetter)
}
