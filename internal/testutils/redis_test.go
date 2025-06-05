package testutils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStartRedis(t *testing.T) {
	container, client, err := StartRedis(t.Context())
	require.NoError(t, err)
	require.NoError(t, client.Ping(t.Context()).Err())
	require.NoError(t, container.Terminate(t.Context()))
}
