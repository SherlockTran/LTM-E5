# TLS Lab: SSL/TLS Integration and Encrypted TCP Tunnel (Go)

This project demonstrates:

- Integrating strong SSL/TLS into client-server apps (echo server/client).
- Building a basic TCP tunnel that can encrypt the upstream leg (client → tunnel → TLS target) or be adapted the other way.
- Generating a development CA, server, and client certificates (PowerShell script).
- Using secure TLS settings: TLS 1.2/1.3, PFS curves, strong cipher suites, optional mTLS.

## Structure

```
cmd/
  echo-server/      # TLS Echo Server
  echo-client/      # TLS Echo Client
  tunnel-server/    # TCP Tunnel that uses TLS upstream
internal/
  tlsutil/          # Shared TLS configs with hardened defaults
scripts/
  gen-certs.ps1     # Windows PowerShell script to generate CA/server/client certs
certs/              # Generated certs (ignored until you run the script)
```

## Prerequisites

- Go 1.22+
- OpenSSL installed and available on PATH (for `gen-certs.ps1`)
- Windows PowerShell (script is Windows-first; users on Linux/macOS can run equivalent OpenSSL commands)

## Generate Certificates (development)

Run in PowerShell from the repo root:

```powershell
.\scripts\gen-certs.ps1 -OutDir certs -CN localhost
```

This creates a dev CA (`ca.crt`/`ca.key`), a server cert for `localhost` (`server.crt`/`server.key`), and a client cert (`client.crt`/`client.key`).

## Build

```powershell
go build ./cmd/echo-server
go build ./cmd/echo-client
go build ./cmd/tunnel-server
```

The binaries are placed next to each `main.go` by default, or use `-o` to specify outputs.

## Run: TLS Echo Server + Client

1) Start server (no mTLS):

```powershell
.\cmd\echo-server\echo-server.exe -addr 0.0.0.0:8443 -cert certs/server.crt -key certs/server.key
```

2) Run client:

```powershell
.\cmd\echo-client\echo-client.exe -addr 127.0.0.1:8443 -servername localhost -ca certs/ca.crt
```

3) mTLS variant: server requires client certificate

```powershell
.\cmd\echo-server\echo-server.exe -addr 0.0.0.0:8443 -cert certs/server.crt -key certs/server.key -mtls -ca certs/ca.crt
.\cmd\echo-client\echo-client.exe -addr 127.0.0.1:8443 -servername localhost -ca certs/ca.crt -cert certs/client.crt -key certs/client.key
```

## Run: TCP Tunnel (TLS upstream)

- Tunnel listens locally in plaintext and connects to the upstream using TLS (default).

```powershell
.\cmd\tunnel-server\tunnel-server.exe -listen 0.0.0.0:8080 -target example.com:443 -target-tls `
  -servername example.com
```

- Then you can connect any TCP client to `127.0.0.1:8080`. For HTTPS testing:

```powershell
# From Windows PowerShell, simple test using OpenSSL s_client via the tunnel:
openssl s_client -connect 127.0.0.1:8080 -crlf
GET / HTTP/1.1
Host: example.com

```

- To tunnel to your local TLS echo server:

```powershell
.\cmd\echo-server\echo-server.exe -addr 127.0.0.1:8443 -cert certs/server.crt -key certs/server.key
.\cmd\tunnel-server\tunnel-server.exe -listen 0.0.0.0:8080 -target 127.0.0.1:8443 -target-tls -servername localhost -ca certs/ca.crt
```

Now any plaintext client connecting to `127.0.0.1:8080` will have its traffic encrypted from the tunnel to the upstream echo server.

## Security Notes

- Minimum TLS version is 1.2; TLS 1.3 is enabled by default.
- Strong ECDHE cipher suites are configured for PFS (Perfect Forward Secrecy).
- Curve preferences prioritize X25519 and P-256.
- Server can be configured to require client certificates (mTLS).
- Use `-servername` on clients to match the certificate’s CN/SAN; when testing locally, use `localhost` and generate certs with SANs.

## Extending

- Swap echo TCP for HTTP, WebSocket, or gRPC servers; reuse `internal/tlsutil` for secure TLS.
- Flip the tunnel direction: accept TLS from client, forward plaintext to target (wrap the `listen` side with TLS).
- Add rate limiting, connection pools, or load balancing for higher throughput.

## Push to GitHub

1) Create a new repo on GitHub (empty, no README).
2) From repo root, run:

```powershell
git init
git add .
git commit -m "Initial commit: TLS echo & tunnel with secure defaults"
git branch -M main
git remote add origin https://github.com/<your-username>/<your-repo>.git
git push -u origin main
```

## Troubleshooting

- If the client fails certificate verification, check `-servername` and that the CA matches the server certificate.
- For mTLS, ensure client cert/key are passed and signed by the same CA the server trusts.
- On Windows, ensure `openssl` is installed and on PATH for the cert script.


