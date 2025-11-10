# TLS Lab: Tích hợp SSL/TLS và TCP Tunnel mã hóa (Go)

Dự án minh họa:

- Tích hợp SSL/TLS an toàn vào ứng dụng client-server (echo server/client).
- Xây dựng TCP tunnel cơ bản, mã hóa lưu lượng ở nhánh upstream (client → tunnel → TLS target) hoặc có thể mở rộng theo chiều ngược lại.
- Sinh chứng chỉ phục vụ phát triển: CA, server, client (PowerShell script).
- Thiết lập TLS an toàn: TLS 1.2/1.3, PFS, cipher suites mạnh, tùy chọn mTLS.

## Cấu trúc

```
cmd/
  echo-server/      # TLS Echo Server
  echo-client/      # TLS Echo Client
  tunnel-server/    # TCP Tunnel dùng TLS ở upstream
internal/
  tlsutil/          # TLS config dùng chung với mặc định an toàn
scripts/
  gen-certs.ps1     # PowerShell sinh CA/server/client certs
certs/              # Thư mục chứa certs (tạo sau khi chạy script)
```

## Yêu cầu

- Go 1.22+
- OpenSSL đã cài và có trong PATH (cho `gen-certs.ps1`)
- Windows PowerShell (script ưu tiên Windows; Linux/macOS dùng lệnh OpenSSL tương đương)

## Tạo chứng chỉ (môi trường phát triển)

Chạy tại thư mục gốc repo (PowerShell):

```powershell
.\scripts\gen-certs.ps1 -OutDir certs -CN localhost
```

Sinh ra CA dev (`ca.crt`/`ca.key`), server cert cho `localhost` (`server.crt`/`server.key`) và client cert (`client.crt`/`client.key`).

## Build

```powershell
go build ./cmd/echo-server
go build ./cmd/echo-client
go build ./cmd/tunnel-server
```

Mặc định, binary được đặt tại thư mục hiện tại (root). Có thể dùng `-o` để chỉ định đầu ra.

## Chạy: TLS Echo Server + Client

1) Khởi động server (không mTLS):

```powershell
.\echo-server.exe -addr 0.0.0.0:8443 -cert certs/server.crt -key certs/server.key
```

2) Chạy client:

```powershell
.\echo-client.exe -addr 127.0.0.1:8443 -servername localhost -ca certs/ca.crt
```

3) Bật mTLS (server yêu cầu client certificate):

```powershell
.\echo-server.exe -addr 0.0.0.0:8443 -cert certs/server.crt -key certs/server.key -mtls -ca certs/ca.crt
.\echo-client.exe -addr 127.0.0.1:8443 -servername localhost -ca certs/ca.crt -cert certs/client.crt -key certs/client.key
```

Lưu ý: Nếu bạn build vào `cmd/...` hoặc `bin/...`, hãy đổi đường dẫn `.exe` tương ứng.

## Chạy: TCP Tunnel (TLS upstream)

- Tunnel lắng nghe local dạng plaintext và kết nối đến upstream bằng TLS (mặc định).

```powershell
.\tunnel-server.exe -listen 0.0.0.0:8080 -target example.com:443 -target-tls -servername example.com
```

- Kết nối bất kỳ TCP client nào vào `127.0.0.1:8080`. Ví dụ test HTTPS qua tunnel:

```powershell
openssl s_client -connect 127.0.0.1:8080 -crlf
GET / HTTP/1.1
Host: example.com

```

- Tunnel đến TLS echo server chạy local:

```powershell
.\echo-server.exe -addr 127.0.0.1:8443 -cert certs/server.crt -key certs/server.key
.\tunnel-server.exe -listen 0.0.0.0:8080 -target 127.0.0.1:8443 -target-tls -servername localhost -ca certs/ca.crt
```

Khi đó, client plaintext vào `127.0.0.1:8080` sẽ được mã hóa từ tunnel đến upstream echo server.

## Ghi chú bảo mật

- Tối thiểu TLS 1.2; bật TLS 1.3 theo mặc định.
- PFS (Perfect Forward Secrecy) với ECDHE; ưu tiên curves X25519, P-256.
- Cipher suites mạnh: AES-GCM, ChaCha20-Poly1305 (TLS 1.3 dùng bộ mặc định an toàn của Go).
- Có thể bật mTLS để xác thực 2 chiều.
- Client nên dùng `-servername` khớp CN/SAN của certificate; test local dùng `localhost` và cert có SAN phù hợp.

## Mở rộng

- Thay echo TCP bằng HTTP, WebSocket, hoặc gRPC; tái sử dụng `internal/tlsutil` cho TLS an toàn.
- Đảo chiều tunnel: nhận TLS từ client, chuyển plaintext đến server đích (bọc TLS phía listen).
- Bổ sung rate limiting, connection pool, hoặc load balancing để chịu tải tốt hơn.

## Khắc phục sự cố

- Client báo lỗi verify cert: kiểm tra `-servername` và CA (`-ca`) có khớp certificate của server.
- mTLS: chắc chắn client cung cấp `-cert/-key` được CA tin cậy của server ký.
- Windows: cần `openssl` trong PATH để chạy script tạo certs.
