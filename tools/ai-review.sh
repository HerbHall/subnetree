#!/usr/bin/env bash
# ai-review.sh -- Send staged git diff to local Ollama for code review.
# Requires: curl, python, Ollama running on localhost:11434
#
# Usage:
#   ./tools/ai-review.sh              # Review staged changes
#   ./tools/ai-review.sh --all        # Review all uncommitted changes
#   OLLAMA_MODEL=llama3.1 ./tools/ai-review.sh  # Use specific model

set -euo pipefail

OLLAMA_HOST="${OLLAMA_HOST:-http://127.0.0.1:11434}"
OLLAMA_MODEL="${OLLAMA_MODEL:-qwen2.5:32b}"

# Find python (Windows Store aliases can shadow real python)
PYTHON=""
for p in python3 python py "/c/Program Files/Python39/python" "/c/Program Files/Python312/python"; do
    if "$p" --version &>/dev/null 2>&1; then PYTHON="$p"; break; fi
done
if [[ -z "$PYTHON" ]]; then echo "Error: python is required." >&2; exit 1; fi

# Check Ollama is reachable
if ! curl -s "${OLLAMA_HOST}/api/tags" &>/dev/null; then
    echo "Error: Ollama not reachable at ${OLLAMA_HOST}" >&2
    echo "Start it with: ollama serve" >&2
    exit 1
fi

# Get the diff
if [[ "${1:-}" == "--all" ]]; then
    diff=$(git diff)
else
    diff=$(git diff --staged)
fi

if [[ -z "$diff" ]]; then
    echo "No changes to review."
    [[ "${1:-}" != "--all" ]] && echo "Tip: stage changes first (git add) or use --all for unstaged."
    exit 0
fi

# Truncate very large diffs
max_chars=12000
if [[ ${#diff} -gt $max_chars ]]; then
    diff="${diff:0:$max_chars}

... [truncated at ${max_chars} chars -- review remaining changes manually]"
fi

echo "Reviewing with ${OLLAMA_MODEL}..."
echo "---"

# Use python for JSON escaping and response parsing
$PYTHON -c "
import json, urllib.request, sys

diff = sys.stdin.read()
prompt = '''You are an expert code reviewer for a Go + React/TypeScript project. Review this diff for:
1. Bugs or logic errors
2. Security issues (injection, XSS, credential leaks)
3. Performance concerns
4. Go/TypeScript best practice violations

Be concise. Only flag real issues, not style preferences. Output a numbered list of findings, or \"No issues found.\" if the code looks good.

Diff:
\`\`\`
''' + diff + '''
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
" <<< "$diff"
