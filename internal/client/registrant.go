package client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/server"
	"log/slog"
	"net/http"
	"sync/atomic"
)

type registrant struct {
	httpClient   *http.Client
	logger       *slog.Logger
	leaderURL    string
	clientURL    string
	cfg          configuration.Configuration
	isRegistered atomic.Bool
}

func (r *registrant) setLeaderURL(host string) {
	r.leaderURL = "http://" + r.cfg.MustURLFromHost(host) + "/leader/register"
}

func (r *registrant) register(ctx context.Context) {
	r.logger.Debug("(re-)registering with leader")
	if r.leaderURL == "" {
		r.logger.Warn("no leader yet. skipping registration request")
		return
	}
	regReq := server.RegistrationRequest{URL: r.clientURL}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(regReq); err != nil {
		r.logger.Error("failed to encode registration request", "err", err)
		return
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, r.leaderURL, &body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("failed to send registration request", "err", err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		r.logger.Error("registration request rejected", "target", r.leaderURL, "status", resp.Status)
		return
	}
	r.isRegistered.Store(true)
}

func (r *registrant) IsRegistered() bool {
	return r.isRegistered.Load()
}
