package registerer_test

import (
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/caller"
	"github.com/clambin/ledswitcher/endpoint/health"
	"github.com/clambin/ledswitcher/endpoint/registerer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRegisterer_Run(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(registryStub))
	defer testServer.Close()

	s, _ := scheduler.New("linear")
	b := broker.New(time.Second, s)
	r := registerer.Registerer{
		Caller:      caller.New(),
		Broker:      b,
		EndPointURL: "http://127.0.0.1:8080",
	}
	r.SetLeaderURL(testServer.URL)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	go func() {
		r.Run(ctx)
	}()

	require.Eventually(t, func() bool { return r.IsRegistered() }, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()
}

func TestRegisterer_Run_Retry(t *testing.T) {
	s, _ := scheduler.New("linear")
	b := broker.New(time.Second, s)
	h := health.Health{}
	r := registerer.Registerer{
		Caller:      caller.New(),
		Broker:      b,
		EndPointURL: "http://127.0.0.1:8080",
		Interval:    150 * time.Millisecond,
		Health:      &h,
	}
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}

	go func() {
		r.Run(ctx)
	}()

	testServer := httptest.NewServer(http.HandlerFunc(registryStub))
	r.SetLeaderURL(testServer.URL)
	require.Eventually(t, func() bool { return r.IsRegistered() }, time.Minute, 100*time.Millisecond)
	assert.True(t, h.IsHealthy())

	testServer.Close()
	require.Eventually(t, func() bool { return !r.IsRegistered() }, time.Minute, 100*time.Millisecond)
	assert.False(t, h.IsHealthy())

	testServer = httptest.NewServer(http.HandlerFunc(registryStub))
	defer testServer.Close()
	r.SetLeaderURL(testServer.URL)
	require.Eventually(t, func() bool { return r.IsRegistered() }, time.Minute, 100*time.Millisecond)
	assert.True(t, h.IsHealthy())

	cancel()
	wg.Wait()
}

func registryStub(w http.ResponseWriter, req *http.Request) {
	defer func() {
		_ = req.Body.Close()
	}()
	if req.Method != http.MethodPost {
		http.Error(w, "wrong method", http.StatusBadRequest)
		return
	}
	if req.URL.Path != "/register" {
		http.Error(w, "endpoint not supported", http.StatusNotFound)
		return
	}

	body, _ := io.ReadAll(req.Body)
	var content interface{}
	err := json.Unmarshal(body, &content)
	if err != nil {
		http.Error(w, "invalid content: "+err.Error(), http.StatusBadRequest)
		return
	}

	value, ok := content.(map[string]interface{})["url"]
	if ok == false || strings.HasPrefix(value.(string), "http://") == false {
		http.Error(w, "invalid content", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
