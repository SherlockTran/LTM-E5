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

## Hướng dẫn cho team: Clone và test nhanh

1) Clone repo (PowerShell/CMD):
   ```powershell
   git clone https://github.com/<your-username>/<your-repo>.git
   cd <your-repo>
   ```

2) Cài đặt yêu cầu tối thiểu
   - Go 1.22+ (khuyến nghị mới nhất)
   - OpenSSL trong PATH (để sinh cert): kiểm tra `openssl version`
   - PowerShell cho Windows

3) Sinh chứng chỉ dev
   ```powershell
   .\scripts\gen-certs.ps1 -OutDir certs -CN localhost
   ```

4) Build ứng dụng cơ bản
   ```powershell
   go build .\cmd\echo-server
   go build .\cmd\echo-client
   go build .\cmd\tunnel-server
   go build .\cmd\grpc-server
   go build .\cmd\grpc-client
   ```

5) Test Echo TLS
   - Server:
     ```powershell
     .\echo-server.exe -addr 0.0.0.0:8443 -cert certs\server.crt -key certs\server.key
     ```
   - Client (cửa sổ khác):
     ```powershell
     .\echo-client.exe -addr 127.0.0.1:8443 -servername localhost -ca certs\ca.crt
     ```

6) Test Tunnel (plaintext → TLS upstream)
   ```powershell
   .\tunnel-server.exe -listen 0.0.0.0:8080 -target example.com:443 -target-tls -servername example.com
   ```

7) Test gRPC (JSON codec, không cần protoc)
   - Server:
     ```powershell
     .\grpc-server.exe -addr 0.0.0.0:9443 -cert certs\server.crt -key certs\server.key
     ```
   - Client:
     ```powershell
     .\grpc-client.exe -addr 127.0.0.1:9443 -servername localhost -ca certs\ca.crt -msg "hello grpc"
     ```

8) gRPC protobuf chuẩn (dùng ghz/grpcurl) – chỉ làm khi cần benchmark protobuf
   - Cài protoc (Windows): tải `protoc-<version>-win64.zip` và thêm `C:\protoc-<ver>\bin` vào PATH
     ```powershell
     $Env:Path = "$Env:Path;C:\protoc-33.0-win64\bin"
     protoc --version
     ```
   - Cài plugin codegen và generate:
     ```powershell
     go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
     go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
     $Env:Path = "$Env:Path;$(go env GOPATH)\bin"
     .\scripts\proto-gen.ps1
     ```
   - Build server/client protobuf:
     ```powershell
     go build .\cmd\grpcpb-server
     go build .\cmd\grpcpb-client
     ```
   - Chạy test protobuf:
     ```powershell
     .\grpcpb-server.exe -addr 0.0.0.0:9443 -cert certs\server.crt -key certs\server.key
     .\grpcpb-client.exe -addr 127.0.0.1:9443 -servername localhost -ca certs\ca.crt -msg "bench"
     ```
   - Cài ghz và benchmark (PowerShell một dòng, target ở cuối):
     ```powershell
     go install github.com/bojand/ghz/cmd/ghz@latest
     $Env:Path = "$Env:Path;$(go env GOPATH)\bin"
     ghz --proto .\api\echo.proto --call echo.Echo.Say -d '{"message":"bench"}' --cacert .\certs\ca.crt --authority localhost 127.0.0.1:9443
     ```
     - CMD tương đương (escape JSON):
       ```cmd
       ghz --proto .\api\echo.proto --call echo.Echo.Say -d "{\"message\":\"bench\"}" --cacert .\certs\ca.crt --authority localhost 127.0.0.1:9443
       ```

9) Nếu PowerShell chặn chạy script (.ps1):
   ```powershell
   Set-ExecutionPolicy -Scope Process Bypass -Force
   ```

10) Nếu lệnh không thấy `.exe` sau khi build: chạy từ thư mục gốc repo với `.\<tên>.exe`, hoặc dùng `go run`:
   ```powershell
   go run .\cmd\echo-server
   ```

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
.\echo-server.exe -addr 0.0.0.0:8443 -cert certs/server.crt -key certs/server.key -pprof 127.0.0.1:6061
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
.\tunnel-server.exe -listen 0.0.0.0:8080 -target example.com:443 -target-tls -servername example.com -pprof 127.0.0.1:6060
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

## gRPC qua TLS (không cần protoc để chạy)

- Server:
  ```powershell
  go build .\cmd\grpc-server
  .\grpc-server.exe -addr 0.0.0.0:9443 -cert certs/server.crt -key certs/server.key
  ```
- Client:
  ```powershell
  go build .\cmd\grpc-client
  .\grpc-client.exe -addr 127.0.0.1:9443 -servername localhost -ca certs/ca.crt -msg "hello grpc"
  ```
- Ghi chú:
  - gRPC dùng JSON codec tự viết để tránh phụ thuộc `protoc` khi chạy demo.
  - Vẫn cung cấp `api/echo.proto` để bạn benchmark với `ghz`.

## gRPC protobuf “chuẩn” (dùng ghz/grpcurl + reflection)

- Generate mã Go từ proto (PowerShell, yêu cầu cài protoc trước – link trong script):
  ```powershell
  .\scripts\proto-gen.ps1
  ```
- Build server/client protobuf:
  ```powershell
  go build .\cmd\grpcpb-server
  go build .\cmd\grpcpb-client
  ```
- Chạy server có reflection:
  ```powershell
  .\grpcpb-server.exe -addr 0.0.0.0:9443 -cert certs\server.crt -key certs\server.key
  ```
- Test client protobuf:
  ```powershell
  .\grpcpb-client.exe -addr 127.0.0.1:9443 -servername localhost -ca certs\ca.crt -msg "bench"
  ```
- Benchmark với ghz (dùng proto, không cần JSON codec):
  ```powershell
  ghz --proto .\api\echo.proto --call echo.Echo.Say -d "{\"message\":\"bench\"}" --cacert .\certs\ca.crt --authority localhost 127.0.0.1:9443
  ```
  Hoặc dùng grpcurl:
  ```powershell
  grpcurl -cacert certs\ca.crt -authority localhost -proto api\echo.proto -d "{\"message\":\"bench\"}" 127.0.0.1:9443 echo.Echo.Say
  ```

## Khắc phục sự cố

- Client báo lỗi verify cert: kiểm tra `-servername` và CA (`-ca`) có khớp certificate của server.
- mTLS: chắc chắn client cung cấp `-cert/-key` được CA tin cậy của server ký.
- Windows: cần `openssl` trong PATH để chạy script tạo certs.

## Quan sát & Benchmark

- pprof: sau khi bật `-pprof`, truy cập:
  - `http://127.0.0.1:6061/debug/pprof/` (echo-server)
  - `http://127.0.0.1:6060/debug/pprof/` (tunnel-server)
- Lấy CPU profile (30s) bằng PowerShell:
  ```powershell
  $out = "cpu.pb"
  Invoke-WebRequest -Uri http://127.0.0.1:6061/debug/pprof/profile?seconds=30 -OutFile $out
  ```
  Mở bằng `go tool pprof`: `go tool pprof -http=:0 $out`

- HTTP(S) benchmark (nếu test upstream HTTPS thật):
  - Dùng `wrk` hoặc `hey`:
    ```powershell
    # ví dụ hey: 100 kết nối đồng thời, 30s
    hey -c 100 -z 30s https://localhost:8443/
    ```
  - Hoặc qua tunnel:
    ```powershell
    hey -c 100 -z 30s http://127.0.0.1:8080/
    ```

- gRPC benchmark (sẽ bổ sung kèm gRPC Echo): dùng `ghz`.
  - Ví dụ (dùng file proto, chỉ benchmark – không cần generate code):
    ```powershell
    ghz --insecure --proto .\api\echo.proto --call echo.Echo.Say `
      -d '{\"message\":\"bench\"}' `
      -H \"authority: localhost\" `
      --cacert .\certs\ca.crt `
      --host override localhost `
      127.0.0.1:9443
    ```
