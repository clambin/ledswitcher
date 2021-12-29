package endpoint_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

func TestEndpoint_LED(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ep, ledSetter, wg := startEndpoint(ctx, 0)

	require.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	ledSetter.On("SetLED", true).Return(nil).Once()
	statusCode, err := doHTTPCall(
		fmt.Sprintf("http://127.0.0.1:%d/led", ep.HTTPServer.Port),
		http.MethodPost,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, statusCode)

	ledSetter.On("SetLED", false).Return(nil).Once()
	statusCode, err = doHTTPCall(
		fmt.Sprintf("http://127.0.0.1:%d/led", ep.HTTPServer.Port),
		http.MethodDelete,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, statusCode)

	ledSetter.On("SetLED", true).Return(nil).Once()
	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, ledSetter)
}

func TestEndpoint_LED_Failure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ep, ledSetter, wg := startEndpoint(ctx, 0)

	require.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	ledSetter.On("SetLED", true).Return(errors.New("failed to set LED")).Once()
	statusCode, err := doHTTPCall(
		ep.MakeURL("127.0.0.1")+"/led",
		http.MethodPost,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, statusCode)

	ledSetter.On("SetLED", true).Return(nil).Once()
	cancel()
	wg.Wait()
	mock.AssertExpectationsForObjects(t, ledSetter)
}
