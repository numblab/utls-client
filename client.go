package client

import (
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

func New() *http.Client {
	return &http.Client{
		Transport: &uTransport{
			H1: &http.Transport{},
			H2: &http2.Transport{
				MaxDecoderHeaderTableSize: 1 << 16,
			},
		},
		Timeout: 30 * time.Second,
	}
}
