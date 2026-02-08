#!/usr/bin/env bash
# verify-release.sh -- Full functional verification of a SubNetree release binary.
#
# Usage:
#   ./verify-release.sh [VERSION] [SUBNET]
#
# Examples:
#   ./verify-release.sh 0.2.1                    # Test on 192.168.1.0/24
#   ./verify-release.sh 0.2.1 10.0.0.0/24        # Test on custom subnet
#   ./verify-release.sh 0.2.1 skip               # Skip network scan
#
# Prerequisites:
#   - curl, tar/unzip
#   - Network access to GitHub releases (for download)
#   - Root/sudo for network scanning (ICMP/ARP require raw sockets)
#
# The script downloads the release binary for the current platform,
# starts the server, runs all functional tests, and produces a report.

set -euo pipefail

VERSION="${1:-0.2.1}"
SUBNET="${2:-192.168.1.0/24}"
PORT=19999
BASE_URL="http://127.0.0.1:${PORT}"
WORKDIR="$(mktemp -d)"
REPORT=""
PASS=0
FAIL=0
WARN=0
SERVER_PID=""

# --- Helpers ---

cleanup() {
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        kill "$SERVER_PID" 2>/dev/null
        wait "$SERVER_PID" 2>/dev/null || true
    fi
    echo ""
    echo "=== Test artifacts in: $WORKDIR ==="
    echo "=== Server log: $WORKDIR/server.log ==="
}
trap cleanup EXIT

log_pass() {
    PASS=$((PASS + 1))
    REPORT="${REPORT}PASS: $1\n"
    echo "[PASS] $1"
}

log_fail() {
    FAIL=$((FAIL + 1))
    REPORT="${REPORT}FAIL: $1\n"
    echo "[FAIL] $1"
}

log_warn() {
    WARN=$((WARN + 1))
    REPORT="${REPORT}WARN: $1\n"
    echo "[WARN] $1"
}

log_info() {
    echo "[INFO] $1"
}

# JSON field extraction without jq (uses Python as fallback)
json_field() {
    local json="$1"
    local field="$2"
    # Try python3, python, then basic grep
    for py in python3 python; do
        if command -v "$py" &>/dev/null && "$py" --version &>/dev/null 2>&1; then
            echo "$json" | "$py" -c "import json,sys; d=json.load(sys.stdin); print(d.get('$field',''))" 2>/dev/null
            return
        fi
    done
    # Fallback: crude regex (handles simple cases)
    echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 | sed 's/.*: *"//;s/"//'
}

# --- Platform detection ---

detect_platform() {
    local os arch ext
    case "$(uname -s)" in
        Linux*)   os="linux" ;;
        Darwin*)  os="darwin" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *)        os="unknown" ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)            arch="unknown" ;;
    esac
    if [ "$os" = "windows" ]; then
        ext="zip"
    else
        ext="tar.gz"
    fi
    echo "${os}_${arch}_${ext}"
}

PLATFORM_INFO=$(detect_platform)
OS=$(echo "$PLATFORM_INFO" | cut -d_ -f1)
ARCH=$(echo "$PLATFORM_INFO" | cut -d_ -f2)
EXT=$(echo "$PLATFORM_INFO" | cut -d_ -f3-)
BINARY="subnetree"
[ "$OS" = "windows" ] && BINARY="subnetree.exe"

log_info "Platform: $OS/$ARCH"
log_info "Version: v$VERSION"
log_info "Working directory: $WORKDIR"
log_info "Test port: $PORT"
echo ""

# --- 1. Download release binary ---

echo "========================================="
echo "  1. DOWNLOAD & EXTRACT"
echo "========================================="

ARCHIVE="subnetree_${VERSION}_${OS}_${ARCH}.${EXT}"
DOWNLOAD_URL="https://github.com/HerbHall/subnetree/releases/download/v${VERSION}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/HerbHall/subnetree/releases/download/v${VERSION}/checksums.txt"

cd "$WORKDIR"

log_info "Downloading $ARCHIVE ..."
if curl -fsSL -o "$ARCHIVE" "$DOWNLOAD_URL"; then
    log_pass "Binary downloaded: $ARCHIVE"
else
    log_fail "Failed to download $ARCHIVE from $DOWNLOAD_URL"
    echo "FATAL: Cannot continue without binary."
    exit 1
fi

# Download checksums
log_info "Downloading checksums.txt ..."
if curl -fsSL -o checksums.txt "$CHECKSUM_URL"; then
    log_pass "Checksums downloaded"
else
    log_warn "Failed to download checksums.txt -- skipping verification"
fi

# Verify checksum
if [ -f checksums.txt ]; then
    if command -v sha256sum &>/dev/null; then
        EXPECTED=$(grep "$ARCHIVE" checksums.txt | awk '{print $1}')
        ACTUAL=$(sha256sum "$ARCHIVE" | awk '{print $1}')
        if [ "$EXPECTED" = "$ACTUAL" ]; then
            log_pass "SHA-256 checksum verified"
        else
            log_fail "Checksum mismatch! Expected: $EXPECTED Got: $ACTUAL"
        fi
    elif command -v shasum &>/dev/null; then
        EXPECTED=$(grep "$ARCHIVE" checksums.txt | awk '{print $1}')
        ACTUAL=$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')
        if [ "$EXPECTED" = "$ACTUAL" ]; then
            log_pass "SHA-256 checksum verified (shasum)"
        else
            log_fail "Checksum mismatch!"
        fi
    else
        log_warn "No sha256sum or shasum available -- skipping checksum verification"
    fi
fi

# Extract
log_info "Extracting ..."
mkdir -p extracted
if [ "$EXT" = "zip" ]; then
    unzip -o "$ARCHIVE" -d extracted >/dev/null 2>&1
else
    tar xzf "$ARCHIVE" -C extracted
fi

if [ -f "extracted/$BINARY" ]; then
    chmod +x "extracted/$BINARY"
    log_pass "Binary extracted: $(ls -lh extracted/$BINARY | awk '{print $5}')"
else
    log_fail "Binary not found after extraction"
    exit 1
fi

echo ""

# --- 2. Version check ---

echo "========================================="
echo "  2. VERSION CHECK"
echo "========================================="

VERSION_OUTPUT=$("./extracted/$BINARY" -version 2>&1 || true)
log_info "Version output: $VERSION_OUTPUT"

if echo "$VERSION_OUTPUT" | grep -q "SubNetree $VERSION"; then
    log_pass "Version string contains $VERSION"
else
    log_fail "Version string does not contain $VERSION"
fi

if echo "$VERSION_OUTPUT" | grep -q "commit:"; then
    log_pass "Commit hash present in version output"
else
    log_fail "Commit hash missing from version output"
fi

if echo "$VERSION_OUTPUT" | grep -q "built:"; then
    log_pass "Build timestamp present in version output"
else
    log_fail "Build timestamp missing from version output"
fi

echo ""

# --- 3. Server startup ---

echo "========================================="
echo "  3. SERVER STARTUP"
echo "========================================="

mkdir -p "$WORKDIR/data"

cat > "$WORKDIR/config.yaml" <<CONF
server:
  host: "127.0.0.1"
  port: $PORT
  data_dir: "$WORKDIR/data"
database:
  path: "$WORKDIR/data/subnetree.db"
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
CONF

log_info "Starting server on port $PORT ..."
# Set vault passphrase via env var to prevent interactive prompt blocking startup.
export SUBNETREE_VAULT_PASSPHRASE="TestVaultPass123!"
"./extracted/$BINARY" -config "$WORKDIR/config.yaml" > "$WORKDIR/server.log" 2>&1 &
SERVER_PID=$!
log_info "Server PID: $SERVER_PID"

# Wait for server to be ready
READY=false
for i in $(seq 1 20); do
    if curl -sf "$BASE_URL/healthz" >/dev/null 2>&1; then
        READY=true
        break
    fi
    sleep 1
done

if [ "$READY" = true ]; then
    log_pass "Server started and healthy (${i}s)"
else
    log_fail "Server failed to start within 20s"
    echo "=== Last 30 lines of server log ==="
    tail -30 "$WORKDIR/server.log" 2>/dev/null || true
    exit 1
fi

echo ""

# --- 4. Health endpoints ---

echo "========================================="
echo "  4. HEALTH ENDPOINTS"
echo "========================================="

HEALTHZ=$(curl -sf "$BASE_URL/healthz" 2>&1 || echo "FAILED")
if [ "$HEALTHZ" != "FAILED" ]; then
    log_pass "/healthz responds"
else
    log_fail "/healthz not responding"
fi

READYZ=$(curl -sf "$BASE_URL/readyz" 2>&1 || echo "FAILED")
if [ "$READYZ" != "FAILED" ]; then
    log_pass "/readyz responds"
else
    log_fail "/readyz not responding"
fi

METRICS_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/metrics" 2>&1)
if [ "$METRICS_STATUS" = "200" ]; then
    log_pass "/metrics (Prometheus) responds 200"
else
    log_warn "/metrics returned HTTP $METRICS_STATUS"
fi

echo ""

# --- 5. Dashboard ---

echo "========================================="
echo "  5. DASHBOARD"
echo "========================================="

DASH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/" 2>&1)
DASH_SIZE=$(curl -s -o /dev/null -w "%{size_download}" "$BASE_URL/" 2>&1)
if [ "$DASH_STATUS" = "200" ]; then
    log_pass "Dashboard serves HTML (HTTP 200, ${DASH_SIZE} bytes)"
else
    log_fail "Dashboard returned HTTP $DASH_STATUS"
fi

DASH_CONTENT=$(curl -s "$BASE_URL/" 2>&1)
if echo "$DASH_CONTENT" | grep -qi "subnetree\|<!DOCTYPE\|<html"; then
    log_pass "Dashboard HTML contains expected content"
else
    log_fail "Dashboard HTML looks wrong"
fi

echo ""

# --- 6. Setup wizard ---

echo "========================================="
echo "  6. SETUP WIZARD (first-run)"
echo "========================================="

SETUP_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/auth/setup" \
    -H "Content-Type: application/json" \
    -d '{"username":"testadmin","email":"test@subnetree.local","password":"TestPass123!"}' 2>&1 || echo "FAILED")

if echo "$SETUP_RESPONSE" | grep -q "testadmin"; then
    log_pass "Setup wizard created admin account"
else
    log_fail "Setup wizard failed: $SETUP_RESPONSE"
fi

echo ""

# --- 7. Authentication ---

echo "========================================="
echo "  7. AUTHENTICATION"
echo "========================================="

LOGIN_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"testadmin","password":"TestPass123!"}' 2>&1 || echo "FAILED")

ACCESS_TOKEN=$(json_field "$LOGIN_RESPONSE" "access_token")
REFRESH_TOKEN=$(json_field "$LOGIN_RESPONSE" "refresh_token")

if [ -n "$ACCESS_TOKEN" ] && [ "$ACCESS_TOKEN" != "" ]; then
    log_pass "Login succeeded, got access token"
else
    log_fail "Login failed: $LOGIN_RESPONSE"
fi

if [ -n "$REFRESH_TOKEN" ] && [ "$REFRESH_TOKEN" != "" ]; then
    log_pass "Refresh token received"
else
    log_warn "No refresh token received"
fi

# Test token refresh
REFRESH_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/auth/refresh" \
    -H "Content-Type: application/json" \
    -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" 2>&1 || echo "FAILED")

NEW_TOKEN=$(json_field "$REFRESH_RESPONSE" "access_token")
if [ -n "$NEW_TOKEN" ] && [ "$NEW_TOKEN" != "" ]; then
    ACCESS_TOKEN="$NEW_TOKEN"
    log_pass "Token refresh succeeded"
else
    log_warn "Token refresh failed: $REFRESH_RESPONSE"
fi

# Test auth rejection
UNAUTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
    "$BASE_URL/api/v1/recon/devices" 2>&1)
if [ "$UNAUTH_STATUS" = "401" ]; then
    log_pass "Unauthenticated request correctly rejected (401)"
else
    log_warn "Unauthenticated request returned $UNAUTH_STATUS (expected 401)"
fi

# Authenticated health endpoint
HEALTH_API=$(curl -sf "$BASE_URL/api/v1/health" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$HEALTH_API" != "FAILED" ]; then
    log_pass "/api/v1/health responds (authenticated)"
else
    log_fail "/api/v1/health not responding"
fi

echo ""

# --- 8. Vault ---

echo "========================================="
echo "  8. CREDENTIAL VAULT"
echo "========================================="

VAULT_STATUS=$(curl -sf "$BASE_URL/api/v1/vault/status" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if echo "$VAULT_STATUS" | grep -q "sealed\|unsealed"; then
    log_pass "Vault status endpoint responds"
else
    log_fail "Vault status failed: $VAULT_STATUS"
fi

# Initialize and unseal
UNSEAL_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/vault/unseal" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -d '{"passphrase":"TestVaultPass123!"}' 2>&1 || echo "FAILED")

if echo "$UNSEAL_RESPONSE" | grep -q "unsealed"; then
    log_pass "Vault initialized and unsealed"
else
    log_fail "Vault unseal failed: $UNSEAL_RESPONSE"
fi

# Create a credential
CRED_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/vault/credentials" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -d '{"name":"test-cred","type":"ssh_password","data":{"username":"testuser","password":"testpass"}}' 2>&1 || echo "FAILED")

CRED_ID=$(json_field "$CRED_RESPONSE" "id")
if [ -n "$CRED_ID" ] && [ "$CRED_ID" != "" ]; then
    log_pass "Credential created: $CRED_ID"
else
    log_fail "Credential creation failed: $CRED_RESPONSE"
fi

# Retrieve (decrypt) credential
if [ -n "$CRED_ID" ] && [ "$CRED_ID" != "" ]; then
    CRED_DATA=$(curl -sf "$BASE_URL/api/v1/vault/credentials/$CRED_ID/data" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")

    if echo "$CRED_DATA" | grep -q "testuser"; then
        log_pass "Credential decrypted successfully"
    else
        log_fail "Credential decryption failed: $CRED_DATA"
    fi
fi

# Seal vault
SEAL_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/vault/seal" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")

if echo "$SEAL_RESPONSE" | grep -q "sealed"; then
    log_pass "Vault sealed"
else
    log_warn "Vault seal response: $SEAL_RESPONSE"
fi

echo ""

# --- 9. Network scan ---

echo "========================================="
echo "  9. NETWORK SCAN"
echo "========================================="

if [ "$SUBNET" = "skip" ]; then
    log_warn "Network scan skipped (pass a subnet as second argument)"
else
    SCAN_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/recon/scan" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -d "{\"subnet\":\"$SUBNET\"}" 2>&1 || echo "FAILED")

    SCAN_ID=$(json_field "$SCAN_RESPONSE" "id")
    if [ -n "$SCAN_ID" ] && [ "$SCAN_ID" != "" ]; then
        log_pass "Scan started: $SCAN_ID"

        # Wait for scan to complete (up to 60s)
        log_info "Waiting for scan to complete (up to 60s) ..."
        for i in $(seq 1 12); do
            sleep 5
            SCAN_STATUS=$(curl -sf "$BASE_URL/api/v1/recon/scans/$SCAN_ID" \
                -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "{}")
            STATUS=$(json_field "$SCAN_STATUS" "status")
            DEVICE_COUNT=$(json_field "$SCAN_STATUS" "device_count")
            log_info "  Status: $STATUS, Devices: $DEVICE_COUNT (${i}0s)"
            if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
                break
            fi
        done

        if [ "$STATUS" = "completed" ]; then
            log_pass "Scan completed"
            if [ "$DEVICE_COUNT" -gt 0 ] 2>/dev/null; then
                log_pass "Discovered $DEVICE_COUNT device(s)"
            else
                log_warn "Scan completed but found 0 devices (may need root/NET_RAW)"
            fi
        else
            log_warn "Scan did not complete within 60s (status: $STATUS)"
        fi
    else
        log_fail "Failed to start scan: $SCAN_RESPONSE"
    fi

    # List devices
    DEVICES=$(curl -sf "$BASE_URL/api/v1/recon/devices" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
    if [ "$DEVICES" != "FAILED" ]; then
        log_pass "Device list endpoint responds"
    else
        log_fail "Device list endpoint failed"
    fi
fi

echo ""

# --- 10. Pulse monitoring ---

echo "========================================="
echo "  10. PULSE MONITORING"
echo "========================================="

CHECKS=$(curl -sf "$BASE_URL/api/v1/pulse/checks" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$CHECKS" != "FAILED" ]; then
    log_pass "Pulse checks endpoint responds"
else
    log_warn "Pulse checks endpoint failed (may not have checks yet)"
fi

ALERTS=$(curl -sf "$BASE_URL/api/v1/pulse/alerts" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$ALERTS" != "FAILED" ]; then
    log_pass "Pulse alerts endpoint responds"
else
    log_warn "Pulse alerts endpoint failed"
fi

echo ""

# --- 11. Insight analytics ---

echo "========================================="
echo "  11. INSIGHT ANALYTICS"
echo "========================================="

ANOMALIES=$(curl -sf "$BASE_URL/api/v1/insight/anomalies" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$ANOMALIES" != "FAILED" ]; then
    log_pass "Insight anomalies endpoint responds"
else
    log_warn "Insight anomalies endpoint failed"
fi

echo ""

# --- 12. Backup ---

echo "========================================="
echo "  12. BACKUP"
echo "========================================="

BACKUP_FILE="$WORKDIR/test-backup.tar.gz"
BACKUP_OUTPUT=$("./extracted/$BINARY" backup --data-dir "$WORKDIR/data" --output "$BACKUP_FILE" 2>&1 || echo "FAILED")

if [ -f "$BACKUP_FILE" ]; then
    BACKUP_SIZE=$(ls -lh "$BACKUP_FILE" | awk '{print $5}')
    log_pass "Backup created: $BACKUP_SIZE"
else
    log_warn "Backup command output: $BACKUP_OUTPUT"
fi

echo ""

# --- Report ---

echo "========================================="
echo "  VERIFICATION REPORT"
echo "========================================="
echo ""
echo "Platform:  $(uname -s) $(uname -m) ($(uname -r))"
echo "Version:   v$VERSION"
echo "Binary:    $ARCHIVE"
echo "Date:      $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo ""
echo "Results:   $PASS passed, $FAIL failed, $WARN warnings"
echo ""

if [ "$FAIL" -gt 0 ]; then
    echo "--- FAILURES ---"
    echo -e "$REPORT" | grep "^FAIL:"
    echo ""
fi

if [ "$WARN" -gt 0 ]; then
    echo "--- WARNINGS ---"
    echo -e "$REPORT" | grep "^WARN:"
    echo ""
fi

echo "--- ALL RESULTS ---"
echo -e "$REPORT"

# Save report to file
echo -e "$REPORT" > "$WORKDIR/report.txt"
echo "Report saved to: $WORKDIR/report.txt"
echo "Server log: $WORKDIR/server.log"

# Exit code
if [ "$FAIL" -gt 0 ]; then
    exit 1
else
    exit 0
fi
