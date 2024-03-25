package chainmux

import (
	"net/http"
	"strings"
)

// Surely there must be an easier way??

var _ http.Handler = ChainMux{}

type ChainMux map[string]http.Handler

func (c ChainMux) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	for path, handler := range c {
		if strings.HasPrefix(request.URL.Path, path) {
			handler.ServeHTTP(writer, request)
			return
		}
	}
	http.NotFound(writer, request)
}
