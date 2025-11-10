Param(
    [string]$OutDir = "certs",
    [string]$CN = "localhost"
)

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
Push-Location $OutDir

Write-Host "Generating CA..."
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.crt -subj "/C=VN/ST=HN/L=HN/O=Edu/OU=Lab/CN=Edu-Lab-CA"

Write-Host "Generating Server key/csr..."
openssl genrsa -out server.key 2048
@"
[ req ]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no
[ req_distinguished_name ]
C = VN
ST = HN
L = HN
O = Edu
OU = Lab
CN = $CN
[ v3_req ]
subjectAltName = @alt_names
[ alt_names ]
DNS.1 = $CN
IP.1 = 127.0.0.1
"@ | Set-Content server_openssl.cnf
openssl req -new -key server.key -out server.csr -config server_openssl.cnf
@"
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = $CN
IP.1 = 127.0.0.1
"@ | Set-Content server_ext.cnf
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -days 825 -sha256 -extfile server_ext.cnf

Write-Host "Generating Client key/csr..."
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj "/C=VN/ST=HN/L=HN/O=Edu/OU=Lab/CN=client"
@"
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
"@ | Set-Content client_ext.cnf
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt -days 825 -sha256 -extfile client_ext.cnf

Write-Host "Done. Files in $OutDir"

Pop-Location


