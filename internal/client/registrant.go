package client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/api"
	"log/slog"
	"net/http"
	"sync/atomic"
)

type Registrant struct {
	leaderURL    string
	clientURL    string
	httpClient   *http.Client
	isRegistered atomic.Bool
	logger       *slog.Logger
}

func (r *Registrant) Register(ctx context.Context) {
	regReq := api.RegistrationRequest{URL: r.clientURL}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(regReq); err != nil {
		r.logger.Error("failed to encode registration request", "err", err)
		return
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, r.leaderURL+"/leader/register", &body)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("failed to send registration request", "err", err)
		return
	}
	if resp.StatusCode != http.StatusCreated {
		r.logger.Error("registration request rejected", "status", resp.Status)
		return
	}
	r.isRegistered.Store(true)
}

func (r *Registrant) IsRegistered() bool {
	return r.isRegistered.Load()
}
