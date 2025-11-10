package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

type ServerTLSOptions struct {
	CertFile           string
	KeyFile            string
	CAFile             string
	RequireClientCert  bool
	MinVersion         uint16
	EnableTLS13        bool
	PreferServerCipher bool
}

type ClientTLSOptions struct {
	CAFile      string
	CertFile    string
	KeyFile     string
	ServerName  string
	MinVersion  uint16
	EnableTLS13 bool
}

// NewServerTLSConfig builds a hardened tls.Config for servers.
func NewServerTLSConfig(opts ServerTLSOptions) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	var clientCAs *x509.CertPool
	clientAuth := tls.NoClientCert
	if opts.RequireClientCert {
		pool := x509.NewCertPool()
		if opts.CAFile != "" {
			b, err := os.ReadFile(opts.CAFile)
			if err != nil {
				return nil, fmt.Errorf("read CA file: %w", err)
			}
			if ok := pool.AppendCertsFromPEM(b); !ok {
				return nil, fmt.Errorf("append CA certs failed")
			}
		}
		clientCAs = pool
		clientAuth = tls.RequireAndVerifyClientCert
	}

	cfg := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		ClientAuth:               clientAuth,
		ClientCAs:                clientCAs,
		PreferServerCipherSuites: opts.PreferServerCipher,
		MinVersion:               opts.MinVersion,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}
	if !opts.EnableTLS13 {
		cfg.MaxVersion = tls.VersionTLS12
	}
	// Strong cipher suites (TLS 1.2); TLS 1.3 suites are fixed by Go runtime.
	cfg.CipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
	return cfg, nil
}

// NewClientTLSConfig builds a hardened tls.Config for clients.
func NewClientTLSConfig(opts ClientTLSOptions) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: opts.MinVersion,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
	}
	if opts.ServerName != "" {
		cfg.ServerName = opts.ServerName
	}
	if !opts.EnableTLS13 {
		cfg.MaxVersion = tls.VersionTLS12
	}
	cfg.CipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	// Trust store
	if opts.CAFile != "" {
		pool := x509.NewCertPool()
		b, err := os.ReadFile(opts.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		if ok := pool.AppendCertsFromPEM(b); !ok {
			return nil, fmt.Errorf("append CA certs failed")
		}
		cfg.RootCAs = pool
	}

	// mTLS (optional)
	if opts.CertFile != "" && opts.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opts.CertFile, opts.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client key pair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}


