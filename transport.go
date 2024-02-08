package client

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	tls "github.com/bogdanfinn/utls"
	"golang.org/x/net/http2"
)

var defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36"

type uTransport struct {
	H1 *http.Transport
	H2 *http2.Transport
	*option
}

func headers(req *http.Request) {
	if defaultUserAgent == "" {
		return
	}
	req.Header.Set("user-agent", defaultUserAgent)
}

func (*uTransport) newSpec() *tls.ClientHelloSpec {
	spec, err := tls.UTLSIdToSpec(tls.HelloChrome_120)
	if err != nil {
		panic(err)
	}
	return &spec
}

func (u *uTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "http" {
		return u.H1.RoundTrip(req)
	} else if req.URL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme: %s", req.URL.Scheme)
	}

	port := req.URL.Port()
	if port == "" {
		port = "443"
	}

	// TCP connection
	tcpAddr := fmt.Sprintf("%s:%s", req.URL.Hostname(), port)
	if addr, ok := u.hosts[tcpAddr]; ok {
		tcpAddr = addr
	}
	conn, err := net.DialTimeout("tcp", tcpAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("net.DialTimeout error: %+v", err)
	}

	// TLS connection
	uConn := tls.UClient(conn, &tls.Config{ServerName: req.URL.Hostname()}, tls.HelloCustom, false, false)
	if err = uConn.ApplyPreset(u.newSpec()); err != nil {
		return nil, fmt.Errorf("uConn.ApplyPreset() error: %+v", err)
	}
	if err = uConn.Handshake(); err != nil {
		return nil, fmt.Errorf("uConn.Handshake() error: %+v", err)
	}

	alpn := uConn.ConnectionState().NegotiatedProtocol
	switch alpn {
	case "h2":
		req.Proto = "HTTP/2.0"
		req.ProtoMajor = 2
		req.ProtoMinor = 0

		headers(req)

		if c, err := u.H2.NewClientConn(uConn); err == nil {
			return c.RoundTrip(req)
		} else {
			return nil, fmt.Errorf("http2.Transport.NewClientConn() error: %+v", err)
		}

	case "http/1.1", "":
		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1

		headers(req)

		if err := req.Write(uConn); err == nil {
			return http.ReadResponse(bufio.NewReader(uConn), req)
		} else {
			return nil, fmt.Errorf("http.Request.Write() error: %+v", err)
		}

	default:
		return nil, fmt.Errorf("unsupported ALPN: %v", alpn)
	}
}
