package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding"

	"tls-lab/internal/grpcjson"
	"tls-lab/internal/tlsutil"
)

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoReply struct {
	Message string `json:"message"`
}

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:9443", "gRPC server address")
		serverName = flag.String("servername", "localhost", "SNI/verify name")
		caFile     = flag.String("ca", "certs/ca.crt", "CA cert (PEM)")
		certFile   = flag.String("cert", "", "Client cert (PEM, optional for mTLS)")
		keyFile    = flag.String("key", "", "Client key (PEM, optional for mTLS)")
		message    = flag.String("msg", "hello grpc", "Message to echo")
		timeout    = flag.Duration("timeout", 5*time.Second, "RPC timeout")
	)
	flag.Parse()

	tcfg, err := tlsutil.NewClientTLSConfig(tlsutil.ClientTLSOptions{
		CAFile:      *caFile,
		CertFile:    *certFile,
		KeyFile:     *keyFile,
		ServerName:  *serverName,
		MinVersion:  tls.VersionTLS12,
		EnableTLS13: true,
	})
	if err != nil {
		log.Fatalf("tls: %v", err)
	}

	cc, err := grpc.Dial(
		*addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tcfg)),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(encoding.GetCodec(grpcjson.Name))),
		grpc.WithBlock(),
		grpc.WithTimeout(*timeout),
	)
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer cc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	req := &EchoRequest{Message: *message}
	var resp EchoReply
	if err := cc.Invoke(ctx, "/echo.Echo/Say", req, &resp); err != nil {
		log.Fatalf("rpc: %v", err)
	}
	log.Printf("reply: %s", resp.Message)
}


