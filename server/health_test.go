package server_test

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

func TestServer_Health(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	assert.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	s := server.New("localhost", 0, time.Second, false, tmpDir)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		err2 := s.Run(ctx)
		wg.Done()
		require.NoError(t, err2)
	}()
	s.Controller.SetLeader(fmt.Sprintf("http://localhost:%d", s.HTTPServer.Port))

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", s.HTTPServer.Port))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"Leader": true,
	"Endpoints": [
		"http://localhost:`)

	_ = resp.Body.Close()

	cancel()
	wg.Wait()
}
