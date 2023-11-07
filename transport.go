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

const UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36"

type uTransport struct {
	H1 *http.Transport
	H2 *http2.Transport
	*option
}

func headers(req *http.Request) {
	req.Header.Set("user-agent", UA)
}

func (*uTransport) newSpec() *tls.ClientHelloSpec {
	return &tls.ClientHelloSpec{
		CipherSuites: []uint16{
			tls.GREASE_PLACEHOLDER,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		CompressionMethods: []uint8{
			tls.CompressionNone,
		},
		Extensions: []tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.PSKKeyExchangeModesExtension{[]uint8{
				tls.PskModeDHE,
			}},
			&tls.SNIExtension{},
			&tls.ALPNExtension{AlpnProtocols: []string{"h2", "http/1.1"}},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
				tls.ECDSAWithP256AndSHA256,
				tls.PSSWithSHA256,
				tls.PKCS1WithSHA256,
				tls.ECDSAWithP384AndSHA384,
				tls.PSSWithSHA384,
				tls.PKCS1WithSHA384,
				tls.PSSWithSHA512,
				tls.PKCS1WithSHA512,
			}},
			&tls.SupportedVersionsExtension{[]uint16{
				tls.GREASE_PLACEHOLDER,
				tls.VersionTLS13,
				tls.VersionTLS12,
			}},
			&tls.ALPSExtension{SupportedProtocols: []string{"h2"}},
			&tls.SupportedCurvesExtension{[]tls.CurveID{
				tls.CurveID(tls.GREASE_PLACEHOLDER),
				tls.X25519,
				tls.CurveP256,
				tls.CurveP384,
			}},
			&tls.UtlsExtendedMasterSecretExtension{},

			&tls.SessionTicketExtension{},
			&tls.UtlsCompressCertExtension{[]tls.CertCompressionAlgo{
				tls.CertCompressionBrotli,
			}},
			&tls.SCTExtension{},
			&tls.StatusRequestExtension{},
			&tls.KeyShareExtension{[]tls.KeyShare{
				{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
				{Group: tls.X25519},
			}},
			&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{
				tls.PointFormatUncompressed,
			}},
			&tls.UtlsGREASEExtension{},
			&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
		},
	}
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
