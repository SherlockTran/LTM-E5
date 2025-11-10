package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"

	"tls-lab/internal/tlsutil"
)

func main() {
	var (
		address           = flag.String("addr", "0.0.0.0:8443", "Listen address")
		certFile          = flag.String("cert", "certs/server.crt", "Server certificate (PEM)")
		keyFile           = flag.String("key", "certs/server.key", "Server private key (PEM)")
		caFile            = flag.String("ca", "certs/ca.crt", "CA cert for client auth (PEM)")
		requireClientCert = flag.Bool("mtls", false, "Require client certificate (mTLS)")
		readTimeout       = flag.Duration("read-timeout", 30*time.Second, "Per-connection read timeout")
		writeTimeout      = flag.Duration("write-timeout", 30*time.Second, "Per-connection write timeout")
	)
	flag.Parse()

	tlsCfg, err := tlsutil.NewServerTLSConfig(tlsutil.ServerTLSOptions{
		CertFile:          *certFile,
		KeyFile:           *keyFile,
		CAFile:            *caFile,
		RequireClientCert: *requireClientCert,
		MinVersion:        tls.VersionTLS12,
		EnableTLS13:       true,
		PreferServerCipher: true,
	})
	if err != nil {
		log.Fatalf("failed to build TLS config: %v", err)
	}

	ln, err := tls.Listen("tcp", *address, tlsCfg)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Printf("TLS Echo Server listening on %s (mTLS=%v)", *address, *requireClientCert)
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		go handleConn(conn, *readTimeout, *writeTimeout)
	}
}

func handleConn(c net.Conn, rt, wt time.Duration) {
	defer c.Close()
	if tlsConn, ok := c.(*tls.Conn); ok {
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("TLS handshake failed: %v", err)
			return
		}
		state := tlsConn.ConnectionState()
		log.Printf("New TLS connection: %s | version=%x | cipher=%x | mTLS=%v",
			c.RemoteAddr().String(),
			state.Version,
			state.CipherSuite,
			len(state.PeerCertificates) > 0,
		)
	}

	buf := make([]byte, 32*1024)
	for {
		_ = c.SetReadDeadline(time.Now().Add(rt))
		n, readErr := c.Read(buf)
		if n > 0 {
			_ = c.SetWriteDeadline(time.Now().Add(wt))
			if _, writeErr := c.Write(buf[:n]); writeErr != nil {
				log.Printf("write error: %v", writeErr)
				return
			}
		}
		if readErr != nil {
			if isEOF(readErr) {
				return
			}
			log.Printf("read error: %v", readErr)
			return
		}
	}
}

func isEOF(err error) bool {
	return err == io.EOF || os.IsTimeout(err)
}


