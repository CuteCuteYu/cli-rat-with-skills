# Build script for server and client

# Build server
go build -o server.exe server/main.go server/command.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Server built successfully" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Server build failed" -ForegroundColor Red
    exit 1
}

# Build client
go build -o client.exe client/main.go
if ($LASTEXITCODE -eq 0) {
    Write-Host "[OK] Client built successfully" -ForegroundColor Green
} else {
    Write-Host "[FAIL] Client build failed" -ForegroundColor Red
    exit 1
}

Write-Host "`nBuild complete!" -ForegroundColor Cyan