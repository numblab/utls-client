package client

import (
	"net/http"
	"testing"
)

func TestClient(t *testing.T) {
	resp, _ := NewClient().Get("https://ascii2d.net/")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		panic(resp.StatusCode)
	}
}
