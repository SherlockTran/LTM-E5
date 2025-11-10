package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof"
	bufpool "tls-lab/internal/pool"
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
		pprofAddr         = flag.String("pprof", "", "pprof listen address (e.g. 127.0.0.1:6061); empty to disable")
	)
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Printf("pprof listening on http://%s/debug/pprof/", *pprofAddr)
			_ = http.ListenAndServe(*pprofAddr, nil)
		}()
	}

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

	// Use pooled buffer and io.Copy with deadlines to reduce allocations
	bufPtr := bufpool.Get()
	defer bufpool.Put(bufPtr)
	buf := *bufPtr

	reader := deadlineReader{c, rt}
	writer := deadlineWriter{c, wt}
	_, _ = io.CopyBuffer(writer, reader, buf)
}

func isEOF(err error) bool {
	return err == io.EOF || os.IsTimeout(err)
}

type deadlineReader struct {
	r  net.Conn
	rt time.Duration
}

func (d deadlineReader) Read(p []byte) (int, error) {
	_ = d.r.SetReadDeadline(time.Now().Add(d.rt))
	return d.r.Read(p)
}

type deadlineWriter struct {
	w  net.Conn
	wt time.Duration
}

func (d deadlineWriter) Write(p []byte) (int, error) {
	_ = d.w.SetWriteDeadline(time.Now().Add(d.wt))
	return d.w.Write(p)
}


