package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"tls-lab/internal/tlsutil"
)

func main() {
	var (
		listenAddr   = flag.String("listen", "0.0.0.0:8080", "Local listen address for tunnel")
		targetAddr   = flag.String("target", "example.com:443", "Upstream server address")
		targetTLS    = flag.Bool("target-tls", true, "Use TLS to connect to upstream target")
		serverName   = flag.String("servername", "", "SNI/verify name for upstream (defaults to host of target)")
		caFile       = flag.String("ca", "certs/ca.crt", "CA cert to trust for upstream (PEM). If empty, use system roots.")
		clientCert   = flag.String("cert", "", "Client cert for upstream mTLS (optional, PEM)")
		clientKey    = flag.String("key", "", "Client key for upstream mTLS (optional, PEM)")
		readTimeout  = flag.Duration("read-timeout", 60*time.Second, "Read deadline per direction")
		writeTimeout = flag.Duration("write-timeout", 60*time.Second, "Write deadline per direction")
	)
	flag.Parse()

	var tlsCfg *tls.Config
	var err error
	if *targetTLS {
		opts := tlsutil.ClientTLSOptions{
			CAFile:      *caFile,
			CertFile:    *clientCert,
			KeyFile:     *clientKey,
			ServerName:  *serverName,
			MinVersion:  tls.VersionTLS12,
			EnableTLS13: true,
		}
		tlsCfg, err = tlsutil.NewClientTLSConfig(opts)
		if err != nil {
			log.Fatalf("failed to build client TLS config: %v", err)
		}
	}

	ln, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	defer ln.Close()
	log.Printf("Tunnel listening on %s -> %s (TLS to target=%v)", *listenAddr, *targetAddr, *targetTLS)

	for {
		clientConn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handle(clientConn, *targetAddr, *targetTLS, tlsCfg, *readTimeout, *writeTimeout)
	}
}

func handle(clientConn net.Conn, target string, targetTLS bool, tlsCfg *tls.Config, rt, wt time.Duration) {
	defer clientConn.Close()

	backendConn, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("connect to target error: %v", err)
		return
	}
	defer backendConn.Close()

	var upstream net.Conn = backendConn
	if targetTLS {
		tconn := tls.Client(backendConn, tlsCfg)
		if err := tconn.Handshake(); err != nil {
			log.Printf("upstream TLS handshake failed: %v", err)
			return
		}
		upstream = tconn
	}
	log.Printf("Tunnel connected %s <-> %s", clientConn.RemoteAddr(), target)

	// Bi-directional copy with deadlines
	errc := make(chan error, 2)
	go proxyWithDeadline(upstream, clientConn, rt, wt, errc) // upstream -> client
	go proxyWithDeadline(clientConn, upstream, rt, wt, errc) // client -> upstream

	<-errc
}

func proxyWithDeadline(dst net.Conn, src net.Conn, rt, wt time.Duration, errc chan<- error) {
	buf := make([]byte, 64*1024)
	for {
		_ = src.SetReadDeadline(time.Now().Add(rt))
		n, rerr := src.Read(buf)
		if n > 0 {
			_ = dst.SetWriteDeadline(time.Now().Add(wt))
			if _, werr := dst.Write(buf[:n]); werr != nil {
				errc <- fmt.Errorf("write: %w", werr)
				return
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				errc <- rerr
				return
			}
			errc <- fmt.Errorf("read: %w", rerr)
			return
		}
	}
}


