package client

import (
	"io"
	"net/http"
	"testing"
)

func TestClient(t *testing.T) {
	resp, _ := New().Get("https://core-api.prod.blur.io/")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(resp.StatusCode)
	}
}

func TestClientWithHost(t *testing.T) {
	resp, err := New(WithHost(map[string]string{
		"ascii2d.net:443": "104.26.4.72:443",
	})).Get("https://ascii2d.net/")
	if err != nil {
		t.Errorf("error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(resp.StatusCode)
	}
}

func TestWithUserAgent(t *testing.T) {
	resp, err := New(WithUserAgent("test")).Get("https://tls.peet.ws/api/all")
	if err != nil {
		t.Errorf("error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	t.Log(string(body))
}
