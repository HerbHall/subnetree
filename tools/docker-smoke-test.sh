#!/usr/bin/env bash
# docker-smoke-test.sh -- Smoke test a locally-built SubNetree Docker image.
#
# Usage:
#   ./tools/docker-smoke-test.sh [IMAGE] [PORT] [CONTAINER_NAME]
#
# Examples:
#   ./tools/docker-smoke-test.sh                              # Defaults
#   ./tools/docker-smoke-test.sh subnetree:local 19998 subnetree-test
#
# CI mode (auto-detected via CI env var, or set --ci flag):
#   CI=true ./tools/docker-smoke-test.sh ghcr.io/herbhall/subnetree:latest
#   ./tools/docker-smoke-test.sh --ci ghcr.io/herbhall/subnetree:latest
#
# Prerequisites:
#   - Docker running
#   - curl available

set -euo pipefail

# Parse --ci flag
CI_MODE="${CI:-false}"
ARGS=()
for arg in "$@"; do
    if [ "$arg" = "--ci" ]; then
        CI_MODE=true
    else
        ARGS+=("$arg")
    fi
done

IMAGE="${ARGS[0]:-subnetree:local}"
PORT="${ARGS[1]:-19998}"
CONTAINER="${ARGS[2]:-subnetree-test}"
BASE_URL="http://127.0.0.1:${PORT}"
PASS=0
FAIL=0
WARN=0
REPORT=""

# --- Helpers ---

cleanup() {
    echo ""
    echo "--- Cleaning up ---"
    docker rm -f "$CONTAINER" 2>/dev/null || true
    docker volume rm subnetree-test-data 2>/dev/null || true
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
    if [ "$CI_MODE" = "true" ]; then
        echo "::error title=Smoke Test Failure::$1"
    fi
}

log_warn() {
    WARN=$((WARN + 1))
    REPORT="${REPORT}WARN: $1\n"
    echo "[WARN] $1"
    if [ "$CI_MODE" = "true" ]; then
        echo "::warning title=Smoke Test Warning::$1"
    fi
}

log_info() {
    echo "[INFO] $1"
}

# JSON field extraction without jq (uses Python as fallback)
json_field() {
    local json="$1"
    local field="$2"
    for py in python3 python "/c/Program Files/Python312/python" "/c/Program Files/Python39/python"; do
        if "$py" --version &>/dev/null 2>&1; then
            echo "$json" | "$py" -c "import json,sys; d=json.load(sys.stdin); print(d.get('$field',''))" 2>/dev/null
            return
        fi
    done
    # Fallback: crude regex
    echo "$json" | grep -o "\"$field\"[[:space:]]*:[[:space:]]*\"[^\"]*\"" | head -1 | sed 's/.*: *"//;s/"//'
}

echo "========================================="
echo "  SubNetree Docker Smoke Test"
echo "========================================="
echo ""
log_info "Image: $IMAGE"
log_info "Port: $PORT"
log_info "Container: $CONTAINER"
echo ""

# --- 1. Start container ---

echo "========================================="
echo "  1. START CONTAINER"
echo "========================================="

# Remove any existing container with the same name
docker rm -f "$CONTAINER" 2>/dev/null || true

docker run -d \
    --name "$CONTAINER" \
    -p "${PORT}:8080" \
    -v subnetree-test-data:/data \
    --cap-add NET_RAW \
    --cap-add NET_ADMIN \
    -e SUBNETREE_VAULT_PASSPHRASE="TestVaultPass123!" \
    -e NV_LOG_LEVEL=debug \
    "$IMAGE"

log_info "Container started, waiting for health check..."

# Wait for server to be ready
READY=false
for i in $(seq 1 30); do
    if curl -sf "$BASE_URL/healthz" >/dev/null 2>&1; then
        READY=true
        break
    fi
    sleep 1
done

if [ "$READY" = true ]; then
    log_pass "Server healthy (${i}s)"
else
    log_fail "Server failed to start within 30s"
    echo "=== Container logs ==="
    docker logs "$CONTAINER" 2>&1 | tail -30
    exit 1
fi

echo ""

# --- 2. Health endpoints ---

echo "========================================="
echo "  2. HEALTH ENDPOINTS"
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
    log_pass "/metrics responds 200"
else
    log_warn "/metrics returned HTTP $METRICS_STATUS"
fi

echo ""

# --- 3. Dashboard ---

echo "========================================="
echo "  3. DASHBOARD"
echo "========================================="

DASH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/" 2>&1)
if [ "$DASH_STATUS" = "200" ]; then
    log_pass "Dashboard serves HTML (HTTP 200)"
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

# --- 4. Setup wizard ---

echo "========================================="
echo "  4. SETUP WIZARD"
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

# --- 5. Authentication ---

echo "========================================="
echo "  5. AUTHENTICATION"
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

echo ""

# --- 6. Vault ---

echo "========================================="
echo "  6. CREDENTIAL VAULT"
echo "========================================="

VAULT_STATUS=$(curl -sf "$BASE_URL/api/v1/vault/status" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if echo "$VAULT_STATUS" | grep -q "sealed\|unsealed"; then
    log_pass "Vault status endpoint responds"
else
    log_fail "Vault status failed: $VAULT_STATUS"
fi

UNSEAL_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/vault/unseal" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -d '{"passphrase":"TestVaultPass123!"}' 2>&1 || echo "FAILED")

if echo "$UNSEAL_RESPONSE" | grep -q "unsealed"; then
    log_pass "Vault initialized and unsealed"
else
    log_fail "Vault unseal failed: $UNSEAL_RESPONSE"
fi

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

if [ -n "$CRED_ID" ] && [ "$CRED_ID" != "" ]; then
    CRED_DATA=$(curl -sf "$BASE_URL/api/v1/vault/credentials/$CRED_ID/data" \
        -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")

    if echo "$CRED_DATA" | grep -q "testuser"; then
        log_pass "Credential decrypted successfully"
    else
        log_fail "Credential decryption failed: $CRED_DATA"
    fi
fi

SEAL_RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/vault/seal" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")

if echo "$SEAL_RESPONSE" | grep -q "sealed"; then
    log_pass "Vault sealed"
else
    log_warn "Vault seal response: $SEAL_RESPONSE"
fi

echo ""

# --- 7. Pulse & Insight ---

echo "========================================="
echo "  7. PULSE & INSIGHT"
echo "========================================="

CHECKS=$(curl -sf "$BASE_URL/api/v1/pulse/checks" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$CHECKS" != "FAILED" ]; then
    log_pass "Pulse checks endpoint responds"
else
    log_warn "Pulse checks endpoint failed"
fi

ALERTS=$(curl -sf "$BASE_URL/api/v1/pulse/alerts" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$ALERTS" != "FAILED" ]; then
    log_pass "Pulse alerts endpoint responds"
else
    log_warn "Pulse alerts endpoint failed"
fi

ANOMALIES=$(curl -sf "$BASE_URL/api/v1/insight/anomalies" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>&1 || echo "FAILED")
if [ "$ANOMALIES" != "FAILED" ]; then
    log_pass "Insight anomalies endpoint responds"
else
    log_warn "Insight anomalies endpoint failed"
fi

echo ""

# --- Report ---

echo "========================================="
echo "  SMOKE TEST REPORT"
echo "========================================="
echo ""
echo "Image:     $IMAGE"
echo "Container: $CONTAINER"
echo "Port:      $PORT"
echo "Date:      $(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date)"
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

# Write GitHub Actions step summary
if [ "$CI_MODE" = "true" ] && [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    {
        echo "## Smoke Test Results"
        echo ""
        echo "| Metric | Count |"
        echo "|--------|-------|"
        echo "| Passed | $PASS |"
        echo "| Failed | $FAIL |"
        echo "| Warnings | $WARN |"
        echo ""
        echo "**Image:** \`$IMAGE\`"
        echo "**Date:** $(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date)"
        echo ""
        if [ "$FAIL" -gt 0 ]; then
            echo "### Failures"
            echo ""
            echo '```'
            echo -e "$REPORT" | grep "^FAIL:"
            echo '```'
            echo ""
        fi
        if [ "$WARN" -gt 0 ]; then
            echo "### Warnings"
            echo ""
            echo '```'
            echo -e "$REPORT" | grep "^WARN:"
            echo '```'
            echo ""
        fi
        echo "<details><summary>Full Results</summary>"
        echo ""
        echo '```'
        echo -e "$REPORT"
        echo '```'
        echo ""
        echo "</details>"
    } >> "$GITHUB_STEP_SUMMARY"
fi

if [ "$FAIL" -gt 0 ]; then
    exit 1
else
    exit 0
fi
