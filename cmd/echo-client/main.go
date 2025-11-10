package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"tls-lab/internal/tlsutil"
)

func main() {
	var (
		address   = flag.String("addr", "127.0.0.1:8443", "Server address")
		serverName = flag.String("servername", "localhost", "ServerName (SNI) to verify")
		caFile    = flag.String("ca", "certs/ca.crt", "CA cert to trust (PEM)")
		certFile  = flag.String("cert", "", "Client certificate (PEM) for mTLS")
		keyFile   = flag.String("key", "", "Client private key (PEM) for mTLS")
		timeout   = flag.Duration("timeout", 10*time.Second, "Dial timeout")
	)
	flag.Parse()

	tlsCfg, err := tlsutil.NewClientTLSConfig(tlsutil.ClientTLSOptions{
		CAFile:     *caFile,
		CertFile:   *certFile,
		KeyFile:    *keyFile,
		ServerName: *serverName,
		MinVersion: tls.VersionTLS12,
		EnableTLS13: true,
	})
	if err != nil {
		log.Fatalf("failed to build TLS config: %v", err)
	}

	dialer := &net.Dialer{Timeout: *timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", *address, tlsCfg)
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	log.Printf("Connected to %s", *address)
	state := conn.ConnectionState()
	log.Printf("TLS version=%x cipher=%x", state.Version, state.CipherSuite)

	fmt.Println("Type messages; Ctrl+C to exit")
	reader := bufio.NewScanner(os.Stdin)
	for reader.Scan() {
		line := reader.Text()
		if _, err := conn.Write([]byte(line + "\n")); err != nil {
			log.Fatalf("write error: %v", err)
		}
		reply, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			log.Fatalf("read error: %v", err)
		}
		fmt.Printf("echo: %s", reply)
	}
	if err := reader.Err(); err != nil {
		log.Fatalf("stdin error: %v", err)
	}
}


