Param(
    [string]$ProtoDir = "api",
    [string]$OutDir = "."
)

Write-Host "Installing protoc-gen-go..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
Write-Host "Installing protoc-gen-go-grpc..."
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Try to find protoc in PATH
$protoc = "protoc"
try {
    $ver = & $protoc --version 2>$null
} catch {
    Write-Host "ERROR: 'protoc' not found in PATH."
    Write-Host "Please install protoc:"
    Write-Host " - Download: https://github.com/protocolbuffers/protobuf/releases"
    Write-Host " - Add protoc.exe to PATH, then rerun this script."
    exit 1
}

Write-Host "Using protoc: $ver"

$env:PATH += ";" + (go env GOPATH) + "\bin"

Write-Host "Generating Go code from .proto..."
& $protoc `
    --go_out=$OutDir `
    --go_opt=paths=source_relative `
    --go-grpc_out=$OutDir `
    --go-grpc_opt=paths=source_relative `
    "$ProtoDir\echo.proto"

if ($LASTEXITCODE -ne 0) {
    Write-Host "protoc failed."
    exit 1
}

# Fix accidental nested path (api\api\echo.pb.go) if present
$nested = Join-Path $OutDir "api\api\echo.pb.go"
$target = Join-Path $OutDir "api\echo.pb.go"
if (Test-Path $nested) {
    New-Item -ItemType Directory -Force -Path (Split-Path $target) | Out-Null
    Move-Item -Force $nested $target
    # Remove empty dir api\api if exists
    $nestDir = Join-Path $OutDir "api\api"
    if (Test-Path $nestDir) {
        Remove-Item $nestDir -Force
    }
}

# Move grpc file similarly if nested or at root
$nestedGrpc = Join-Path $OutDir "api\api\echo_grpc.pb.go"
$targetGrpc = Join-Path $OutDir "api\echo\echo_grpc.pb.go"
if (Test-Path $nestedGrpc) {
    New-Item -ItemType Directory -Force -Path (Split-Path $targetGrpc) | Out-Null
    Move-Item -Force $nestedGrpc $targetGrpc
}
$rootGrpc = Join-Path $OutDir "api\echo_grpc.pb.go"
if (Test-Path $rootGrpc) {
    New-Item -ItemType Directory -Force -Path (Split-Path $targetGrpc) | Out-Null
    Move-Item -Force $rootGrpc $targetGrpc
}
# Ensure echo.pb.go also under api/echo if generated at root
$rootMsg = Join-Path $OutDir "api\echo.pb.go"
if (Test-Path $rootMsg) {
    $msgTarget = Join-Path $OutDir "api\echo\echo.pb.go"
    New-Item -ItemType Directory -Force -Path (Split-Path $msgTarget) | Out-Null
    Move-Item -Force $rootMsg $msgTarget
}

Write-Host "Done. Generated files in $OutDir"


