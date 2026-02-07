#!/usr/bin/env bash
# ai-test.sh -- Generate Go test skeletons for a file using local Ollama.
# Requires: curl, python, Ollama running on localhost:11434
#
# Usage:
#   ./tools/ai-test.sh internal/recon/scanner.go
#   OLLAMA_MODEL=llama3.1 ./tools/ai-test.sh pkg/llm/provider.go

set -euo pipefail

OLLAMA_HOST="${OLLAMA_HOST:-http://127.0.0.1:11434}"
OLLAMA_MODEL="${OLLAMA_MODEL:-qwen2.5:32b}"

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <go-file>" >&2
    exit 1
fi

file="$1"
if [[ ! -f "$file" ]]; then
    echo "Error: File not found: $file" >&2
    exit 1
fi

# Find python (Windows Store aliases can shadow real python)
PYTHON=""
for p in python3 python py "/c/Program Files/Python39/python" "/c/Program Files/Python312/python"; do
    if "$p" --version &>/dev/null 2>&1; then PYTHON="$p"; break; fi
done
if [[ -z "$PYTHON" ]]; then echo "Error: python is required." >&2; exit 1; fi

# Check Ollama is reachable
if ! curl -s "${OLLAMA_HOST}/api/tags" &>/dev/null; then
    echo "Error: Ollama not reachable at ${OLLAMA_HOST}" >&2
    exit 1
fi

echo "Generating tests for ${file} with ${OLLAMA_MODEL}..."
echo "---"

$PYTHON -c "
import json, urllib.request, sys

content = sys.stdin.read()
prompt = '''You are an expert Go developer. Generate table-driven test skeletons for the exported functions in this Go file. Follow these conventions:
- Use testing.T and subtests (t.Run)
- Use table-driven tests with []struct{name string; ...}
- Include edge cases and error cases
- Do NOT mock external dependencies -- just create the test structure
- Output ONLY the Go test code, no explanation

File: ${file}

\`\`\`go
''' + content + '''
\`\`\`'''

payload = json.dumps({'model': '${OLLAMA_MODEL}', 'prompt': prompt, 'stream': False}).encode()
req = urllib.request.Request('${OLLAMA_HOST}/api/generate', data=payload, headers={'Content-Type': 'application/json'})
try:
    resp = urllib.request.urlopen(req, timeout=300)
    result = json.loads(resp.read())
    print(result.get('response', 'Error: No response from model'))
except Exception as e:
    print(f'Error: {e}', file=sys.stderr)
    sys.exit(1)
" < "$file"
