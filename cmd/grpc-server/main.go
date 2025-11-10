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

	"tls-lab/internal/grpcjson"
	"tls-lab/internal/tlsutil"
)

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoReply struct {
	Message string `json:"message"`
}

type EchoServer interface {
	Say(context.Context, *EchoRequest) (*EchoReply, error)
}

type echoServerImpl struct{}

func (s *echoServerImpl) Say(ctx context.Context, req *EchoRequest) (*EchoReply, error) {
	return &EchoReply{Message: req.Message}, nil
}

var _EchoServiceDesc = grpc.ServiceDesc{
	ServiceName: "echo.Echo",
	HandlerType: (*EchoServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Say",
			Handler:    _Echo_Say_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/echo.proto",
}

func _Echo_Say_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EchoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EchoServer).Say(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/echo.Echo/Say",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EchoServer).Say(ctx, req.(*EchoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func main() {
	var (
		addr      = flag.String("addr", "0.0.0.0:9443", "gRPC listen address")
		certFile  = flag.String("cert", "certs/server.crt", "Server cert (PEM)")
		keyFile   = flag.String("key", "certs/server.key", "Server key (PEM)")
		caFile    = flag.String("ca", "certs/ca.crt", "Client CA for mTLS (optional)")
		mtls      = flag.Bool("mtls", false, "Require client certs (mTLS)")
		pprofAddr = flag.String("pprof", "", "pprof listen address (e.g. 127.0.0.1:6062); empty to disable")
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

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tcfg)),
	)
	// Ensure JSON codec is registered
	_ = grpcjson.Name

	grpcServer.RegisterService(&_EchoServiceDesc, &echoServerImpl{})
	log.Printf("gRPC Echo Server on %s (mTLS=%v)", *addr, *mtls)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}


