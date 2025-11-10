package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"

	_ "net/http/pprof"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	"tls-lab/api/echo"
	"tls-lab/internal/tlsutil"
)

type echoServer struct {
	echo.UnimplementedEchoServer
}

func (s *echoServer) Say(_ context.Context, in *echo.EchoRequest) (*echo.EchoReply, error) {
	return &echo.EchoReply{Message: in.GetMessage()}, nil
}

func main() {
	var (
		addr      = flag.String("addr", "0.0.0.0:9443", "gRPC listen address")
		certFile  = flag.String("cert", "certs/server.crt", "Server cert (PEM)")
		keyFile   = flag.String("key", "certs/server.key", "Server key (PEM)")
		caFile    = flag.String("ca", "certs/ca.crt", "Client CA for mTLS (optional)")
		mtls      = flag.Bool("mtls", false, "Require client certs (mTLS)")
		pprofAddr = flag.String("pprof", "", "pprof listen address (e.g. 127.0.0.1:6063); empty to disable")
	)
	flag.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Printf("pprof listening on http://%s/debug/pprof/", *pprofAddr)
			_ = http.ListenAndServe(*pprofAddr, nil)
		}()
	}

	tcfg, err := tlsutil.NewServerTLSConfig(tlsutil.ServerTLSOptions{
		CertFile:           *certFile,
		KeyFile:            *keyFile,
		CAFile:             *caFile,
		RequireClientCert:  *mtls,
		MinVersion:         tls.VersionTLS12,
		EnableTLS13:        true,
		PreferServerCipher: true,
	})
	if err != nil {
		log.Fatalf("tls: %v", err)
	}

	lis, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tcfg)))
	echo.RegisterEchoServer(grpcServer, &echoServer{})
	reflection.Register(grpcServer)

	log.Printf("gRPC PB Echo Server on %s (mTLS=%v)", *addr, *mtls)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}


