# verify-release.ps1 -- Full functional verification of a SubNetree release binary.
#
# Usage:
#   .\verify-release.ps1 [-Version "0.2.1"] [-Subnet "192.168.1.0/24"]
#   .\verify-release.ps1 -Version "0.2.1" -Subnet "skip"
#
# Prerequisites:
#   - PowerShell 5.1+ (built into Windows)
#   - Administrator for network scanning (ICMP/ARP require raw sockets)

param(
    [string]$Version = "0.2.1",
    [string]$Subnet = "192.168.1.0/24",
    [int]$Port = 19999
)

$ErrorActionPreference = "Stop"
$BaseUrl = "http://127.0.0.1:$Port"
$WorkDir = Join-Path $env:TEMP "subnetree-verify-$(Get-Date -Format 'yyyyMMddHHmmss')"
$Pass = 0
$Fail = 0
$Warn = 0
$Report = @()
$ServerProcess = $null

function Log-Pass($msg) { $script:Pass++; $script:Report += "PASS: $msg"; Write-Host "[PASS] $msg" -ForegroundColor Green }
function Log-Fail($msg) { $script:Fail++; $script:Report += "FAIL: $msg"; Write-Host "[FAIL] $msg" -ForegroundColor Red }
function Log-Warn($msg) { $script:Warn++; $script:Report += "WARN: $msg"; Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Log-Info($msg) { Write-Host "[INFO] $msg" -ForegroundColor Cyan }

function Invoke-Api {
    param([string]$Method = "GET", [string]$Uri, [string]$Body, [string]$Token)
    $headers = @{ "Content-Type" = "application/json" }
    if ($Token) { $headers["Authorization"] = "Bearer $Token" }
    try {
        $params = @{ Uri = $Uri; Method = $Method; Headers = $headers; UseBasicParsing = $true }
        if ($Body) { $params["Body"] = $Body }
        $response = Invoke-WebRequest @params
        return $response.Content | ConvertFrom-Json
    } catch {
        return $null
    }
}

try {
    New-Item -ItemType Directory -Path $WorkDir -Force | Out-Null
    New-Item -ItemType Directory -Path "$WorkDir\data" -Force | Out-Null

    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "arm64" }
    # Detect ARM
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { $arch = "arm64" }
    $Archive = "subnetree_${Version}_windows_${arch}.zip"
    $Binary = "subnetree.exe"

    Log-Info "Platform: Windows $arch ($([Environment]::OSVersion.Version))"
    Log-Info "Version: v$Version"
    Log-Info "Working directory: $WorkDir"
    Log-Info "Test port: $Port"
    Write-Host ""

    # ===== 1. DOWNLOAD & EXTRACT =====
    Write-Host "========================================="
    Write-Host "  1. DOWNLOAD & EXTRACT"
    Write-Host "========================================="

    $DownloadUrl = "https://github.com/HerbHall/subnetree/releases/download/v${Version}/${Archive}"
    $ChecksumUrl = "https://github.com/HerbHall/subnetree/releases/download/v${Version}/checksums.txt"
    $ArchivePath = Join-Path $WorkDir $Archive

    Log-Info "Downloading $Archive ..."
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $ArchivePath -UseBasicParsing
        Log-Pass "Binary downloaded: $Archive"
    } catch {
        Log-Fail "Failed to download $Archive"
        exit 1
    }

    # Download checksums
    try {
        Invoke-WebRequest -Uri $ChecksumUrl -OutFile "$WorkDir\checksums.txt" -UseBasicParsing
        Log-Pass "Checksums downloaded"

        # Verify checksum
        $expected = (Get-Content "$WorkDir\checksums.txt" | Where-Object { $_ -match $Archive }) -replace '\s+.*$',''
        $actual = (Get-FileHash $ArchivePath -Algorithm SHA256).Hash.ToLower()
        if ($expected -eq $actual) {
            Log-Pass "SHA-256 checksum verified"
        } else {
            Log-Fail "Checksum mismatch! Expected: $expected Got: $actual"
        }
    } catch {
        Log-Warn "Could not verify checksums"
    }

    # Extract
    Log-Info "Extracting ..."
    $ExtractDir = "$WorkDir\extracted"
    Expand-Archive -Path $ArchivePath -DestinationPath $ExtractDir -Force
    $BinaryPath = Join-Path $ExtractDir $Binary

    if (Test-Path $BinaryPath) {
        $size = (Get-Item $BinaryPath).Length / 1MB
        Log-Pass "Binary extracted: $([math]::Round($size, 1)) MB"
    } else {
        Log-Fail "Binary not found after extraction"
        exit 1
    }

    Write-Host ""

    # ===== 2. VERSION CHECK =====
    Write-Host "========================================="
    Write-Host "  2. VERSION CHECK"
    Write-Host "========================================="

    $VersionOutput = & $BinaryPath -version 2>&1 | Out-String
    Log-Info "Version output: $($VersionOutput.Trim())"

    if ($VersionOutput -match "SubNetree $Version") { Log-Pass "Version string contains $Version" }
    else { Log-Fail "Version string does not contain $Version" }

    if ($VersionOutput -match "commit:") { Log-Pass "Commit hash present" }
    else { Log-Fail "Commit hash missing" }

    if ($VersionOutput -match "built:") { Log-Pass "Build timestamp present" }
    else { Log-Fail "Build timestamp missing" }

    Write-Host ""

    # ===== 3. SERVER STARTUP =====
    Write-Host "========================================="
    Write-Host "  3. SERVER STARTUP"
    Write-Host "========================================="

    $ConfigPath = "$WorkDir\config.yaml"
    $DataDir = "$WorkDir\data"
    $DbPath = "$DataDir\subnetree.db"
    @"
server:
  host: "127.0.0.1"
  port: $Port
  data_dir: "$($DataDir -replace '\\','\\')"
database:
  path: "$($DbPath -replace '\\','\\')"
logging:
  level: "debug"
  format: "console"
plugins:
  recon:
    enabled: true
  pulse:
    enabled: true
  dispatch:
    enabled: true
  vault:
    enabled: true
  gateway:
    enabled: true
  llm:
    enabled: false
  insight:
    enabled: true
"@ | Set-Content $ConfigPath

    Log-Info "Starting server on port $Port ..."
    # Set vault passphrase via env var to prevent interactive prompt blocking startup.
    $env:SUBNETREE_VAULT_PASSPHRASE = "TestVaultPass123!"
    $ServerProcess = Start-Process -FilePath $BinaryPath -ArgumentList "-config",$ConfigPath `
        -RedirectStandardOutput "$WorkDir\server-stdout.log" `
        -RedirectStandardError "$WorkDir\server-stderr.log" `
        -PassThru -WindowStyle Hidden

    Log-Info "Server PID: $($ServerProcess.Id)"

    # Wait for server to be ready
    $ready = $false
    for ($i = 1; $i -le 20; $i++) {
        Start-Sleep -Seconds 1
        try {
            $null = Invoke-WebRequest -Uri "$BaseUrl/healthz" -UseBasicParsing -TimeoutSec 2
            $ready = $true
            break
        } catch { }
    }

    if ($ready) { Log-Pass "Server started and healthy (${i}s)" }
    else {
        Log-Fail "Server failed to start within 20s"
        if (Test-Path "$WorkDir\server-stderr.log") {
            Write-Host "=== Server error log ==="
            Get-Content "$WorkDir\server-stderr.log" -Tail 30
        }
        exit 1
    }

    Write-Host ""

    # ===== 4. HEALTH ENDPOINTS =====
    Write-Host "========================================="
    Write-Host "  4. HEALTH ENDPOINTS"
    Write-Host "========================================="

    foreach ($ep in @("/healthz", "/readyz")) {
        try {
            $null = Invoke-WebRequest -Uri "$BaseUrl$ep" -UseBasicParsing -TimeoutSec 5
            Log-Pass "$ep responds (public)"
        } catch { Log-Fail "$ep not responding" }
    }

    try {
        $metricsResp = Invoke-WebRequest -Uri "$BaseUrl/metrics" -UseBasicParsing -TimeoutSec 5
        if ($metricsResp.StatusCode -eq 200) { Log-Pass "/metrics (Prometheus) responds 200" }
        else { Log-Warn "/metrics returned $($metricsResp.StatusCode)" }
    } catch { Log-Warn "/metrics failed" }

    Write-Host ""

    # ===== 5. DASHBOARD =====
    Write-Host "========================================="
    Write-Host "  5. DASHBOARD"
    Write-Host "========================================="

    try {
        $dashResp = Invoke-WebRequest -Uri "$BaseUrl/" -UseBasicParsing -TimeoutSec 5
        Log-Pass "Dashboard serves HTML (HTTP $($dashResp.StatusCode), $($dashResp.Content.Length) bytes)"
        if ($dashResp.Content -match "(?i)subnetree|<!DOCTYPE|<html") {
            Log-Pass "Dashboard HTML contains expected content"
        } else { Log-Fail "Dashboard HTML looks wrong" }
    } catch { Log-Fail "Dashboard failed to load" }

    Write-Host ""

    # ===== 6. SETUP WIZARD =====
    Write-Host "========================================="
    Write-Host "  6. SETUP WIZARD (first-run)"
    Write-Host "========================================="

    $setupResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/auth/setup" `
        -Body '{"username":"testadmin","email":"test@subnetree.local","password":"TestPass123!"}'

    if ($setupResult -and $setupResult.username -eq "testadmin") { Log-Pass "Setup wizard created admin account" }
    else { Log-Fail "Setup wizard failed" }

    Write-Host ""

    # ===== 7. AUTHENTICATION =====
    Write-Host "========================================="
    Write-Host "  7. AUTHENTICATION"
    Write-Host "========================================="

    $loginResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/auth/login" `
        -Body '{"username":"testadmin","password":"TestPass123!"}'

    $accessToken = $loginResult.access_token
    $refreshToken = $loginResult.refresh_token

    if ($accessToken) { Log-Pass "Login succeeded, got access token" }
    else { Log-Fail "Login failed" }

    if ($refreshToken) { Log-Pass "Refresh token received" }
    else { Log-Warn "No refresh token received" }

    # Token refresh
    $refreshResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/auth/refresh" `
        -Body "{`"refresh_token`":`"$refreshToken`"}"
    if ($refreshResult -and $refreshResult.access_token) {
        $accessToken = $refreshResult.access_token
        Log-Pass "Token refresh succeeded"
    } else { Log-Warn "Token refresh failed" }

    # Auth rejection test
    try {
        $null = Invoke-WebRequest -Uri "$BaseUrl/api/v1/recon/devices" -UseBasicParsing -TimeoutSec 5
        Log-Warn "Unauthenticated request was NOT rejected"
    } catch {
        if ($_.Exception.Response.StatusCode.Value__ -eq 401) {
            Log-Pass "Unauthenticated request correctly rejected (401)"
        } else { Log-Warn "Unexpected status: $($_.Exception.Response.StatusCode.Value__)" }
    }

    # Authenticated health endpoint
    $healthResult = Invoke-Api -Uri "$BaseUrl/api/v1/health" -Token $accessToken
    if ($null -ne $healthResult) { Log-Pass "/api/v1/health responds (authenticated)" }
    else { Log-Fail "/api/v1/health not responding" }

    Write-Host ""

    # ===== 8. VAULT =====
    Write-Host "========================================="
    Write-Host "  8. CREDENTIAL VAULT"
    Write-Host "========================================="

    $vaultStatus = Invoke-Api -Uri "$BaseUrl/api/v1/vault/status" -Token $accessToken
    if ($null -ne $vaultStatus) { Log-Pass "Vault status endpoint responds" }
    else { Log-Fail "Vault status failed" }

    $unsealResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/vault/unseal" `
        -Body '{"passphrase":"TestVaultPass123!"}' -Token $accessToken
    if ($unsealResult -and $unsealResult.status -eq "unsealed") { Log-Pass "Vault initialized and unsealed" }
    else { Log-Fail "Vault unseal failed" }

    $credResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/vault/credentials" `
        -Body '{"name":"test-cred","type":"ssh_password","data":{"username":"testuser","password":"testpass"}}' `
        -Token $accessToken
    $credId = $credResult.id

    if ($credId) { Log-Pass "Credential created: $credId" }
    else { Log-Fail "Credential creation failed" }

    if ($credId) {
        $credData = Invoke-Api -Uri "$BaseUrl/api/v1/vault/credentials/$credId/data" -Token $accessToken
        if ($credData -and $credData.data.username -eq "testuser") { Log-Pass "Credential decrypted successfully" }
        else { Log-Fail "Credential decryption failed" }
    }

    $sealResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/vault/seal" -Token $accessToken
    if ($sealResult -and $sealResult.status -eq "sealed") { Log-Pass "Vault sealed" }
    else { Log-Warn "Vault seal unexpected response" }

    Write-Host ""

    # ===== 9. NETWORK SCAN =====
    Write-Host "========================================="
    Write-Host "  9. NETWORK SCAN"
    Write-Host "========================================="

    if ($Subnet -eq "skip") {
        Log-Warn "Network scan skipped"
    } else {
        $scanResult = Invoke-Api -Method POST -Uri "$BaseUrl/api/v1/recon/scan" `
            -Body "{`"subnet`":`"$Subnet`"}" -Token $accessToken
        $scanId = $scanResult.id

        if ($scanId) {
            Log-Pass "Scan started: $scanId"

            Log-Info "Waiting for scan to complete (up to 60s) ..."
            $scanStatus = "running"
            $deviceCount = 0
            for ($i = 1; $i -le 12; $i++) {
                Start-Sleep -Seconds 5
                $scanCheck = Invoke-Api -Uri "$BaseUrl/api/v1/recon/scans/$scanId" -Token $accessToken
                if ($scanCheck) {
                    $scanStatus = $scanCheck.status
                    $deviceCount = [int]$scanCheck.total
                    Log-Info "  Status: $scanStatus, Devices: $deviceCount ($($i*5)s)"
                }
                if ($scanStatus -eq "completed" -or $scanStatus -eq "failed") { break }
            }

            if ($scanStatus -eq "completed") {
                Log-Pass "Scan completed"
                if ($deviceCount -gt 0) { Log-Pass "Discovered $deviceCount device(s)" }
                else { Log-Warn "Scan completed but found 0 devices (may need Administrator)" }
            } else { Log-Warn "Scan did not complete within 60s (status: $scanStatus)" }
        } else { Log-Fail "Failed to start scan" }
    }

    Write-Host ""

    # ===== 10. PULSE MONITORING =====
    Write-Host "========================================="
    Write-Host "  10. PULSE MONITORING"
    Write-Host "========================================="

    # Use raw HTTP check -- ConvertFrom-Json drops empty arrays in PS5.1
    foreach ($ep in @("checks", "alerts")) {
        try {
            $resp = Invoke-WebRequest -Uri "$BaseUrl/api/v1/pulse/$ep" `
                -Headers @{ "Authorization" = "Bearer $accessToken" } -UseBasicParsing -TimeoutSec 5
            if ($resp.StatusCode -eq 200) { Log-Pass "Pulse $ep endpoint responds (HTTP 200)" }
            else { Log-Warn "Pulse $ep returned HTTP $($resp.StatusCode)" }
        } catch { Log-Warn "Pulse $ep endpoint failed" }
    }

    Write-Host ""

    # ===== 11. INSIGHT ANALYTICS =====
    Write-Host "========================================="
    Write-Host "  11. INSIGHT ANALYTICS"
    Write-Host "========================================="

    # Use raw HTTP check -- ConvertFrom-Json drops empty arrays in PS5.1
    try {
        $resp = Invoke-WebRequest -Uri "$BaseUrl/api/v1/insight/anomalies" `
            -Headers @{ "Authorization" = "Bearer $accessToken" } -UseBasicParsing -TimeoutSec 5
        if ($resp.StatusCode -eq 200) { Log-Pass "Insight anomalies endpoint responds (HTTP 200)" }
        else { Log-Warn "Insight anomalies returned HTTP $($resp.StatusCode)" }
    } catch { Log-Warn "Insight anomalies endpoint failed" }

    Write-Host ""

} finally {
    # Cleanup
    if ($ServerProcess -and !$ServerProcess.HasExited) {
        Stop-Process -Id $ServerProcess.Id -Force -ErrorAction SilentlyContinue
        Log-Info "Server stopped"
    }
}

# ===== REPORT =====
Write-Host "========================================="
Write-Host "  VERIFICATION REPORT"
Write-Host "========================================="
Write-Host ""
Write-Host "Platform:  Windows $arch ($([Environment]::OSVersion.VersionString))"
Write-Host "Version:   v$Version"
Write-Host "Binary:    $Archive"
Write-Host "Date:      $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')"
Write-Host ""
Write-Host "Results:   $Pass passed, $Fail failed, $Warn warnings"
Write-Host ""

if ($Fail -gt 0) {
    Write-Host "--- FAILURES ---" -ForegroundColor Red
    $Report | Where-Object { $_ -match "^FAIL:" } | ForEach-Object { Write-Host $_ -ForegroundColor Red }
    Write-Host ""
}

if ($Warn -gt 0) {
    Write-Host "--- WARNINGS ---" -ForegroundColor Yellow
    $Report | Where-Object { $_ -match "^WARN:" } | ForEach-Object { Write-Host $_ -ForegroundColor Yellow }
    Write-Host ""
}

Write-Host "--- ALL RESULTS ---"
$Report | ForEach-Object { Write-Host $_ }

# Save report
$Report | Set-Content "$WorkDir\report.txt"
Write-Host ""
Write-Host "Report saved to: $WorkDir\report.txt"
Write-Host "Server log: $WorkDir\server-stdout.log"
Write-Host "Server errors: $WorkDir\server-stderr.log"

if ($Fail -gt 0) { exit 1 } else { exit 0 }
