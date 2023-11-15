package client

import (
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

func New(fnList ...OptionFn) *http.Client {
	o := &option{hosts: make(map[string]string)}
	for _, fn := range fnList {
		fn(o)
	}
	return &http.Client{
		Transport: &uTransport{
			H1: &http.Transport{},
			H2: &http2.Transport{
				MaxHeaderListSize:         262144,
				MaxDecoderHeaderTableSize: 65536,
				MaxEncoderHeaderTableSize: 65536,
			},
			option: o,
		},
		Timeout: 30 * time.Second,
	}
}
