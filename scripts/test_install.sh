#!/usr/bin/env bash
# Regression test: ensure every GitHub-facing curl call in install.sh
# uses the shared retry/timeout flags defined in curl_opts.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_SH="$SCRIPT_DIR/install.sh"

if [ ! -f "$INSTALL_SH" ]; then
  echo "FAIL: install.sh not found at $INSTALL_SH" >&2
  exit 1
fi

errors=0

# 1. Verify curl_opts is defined with the required flags
required_flags="--connect-timeout --retry --retry-delay --retry-max-time --retry-all-errors"
for flag in $required_flags; do
  if ! grep -q "^curl_opts=.*${flag}" "$INSTALL_SH"; then
    echo "FAIL: curl_opts is missing '$flag'" >&2
    errors=$((errors + 1))
  fi
done

# 2. Every real curl invocation in install.sh must reference $curl_opts so the
#    shared retry/timeout flags are applied consistently across downloads.
curl_calls=$(grep -n '^[[:space:]]*curl[[:space:]]' "$INSTALL_SH" || true)

if [ -z "$curl_calls" ]; then
  echo "FAIL: no curl calls found in install.sh — test may be stale" >&2
  exit 1
fi

count_total=0
count_with_opts=0
while IFS= read -r line; do
  count_total=$((count_total + 1))
  if echo "$line" | grep -Eq '\$curl_opts|\$\{curl_opts\}'; then
    count_with_opts=$((count_with_opts + 1))
  else
    lineno=$(echo "$line" | cut -d: -f1)
    echo "FAIL: curl call at line $lineno does not use \$curl_opts" >&2
    errors=$((errors + 1))
  fi
done <<< "$curl_calls"

if [ "$count_total" -ne 3 ]; then
  echo "FAIL: expected 3 curl calls in install.sh, found $count_total" >&2
  errors=$((errors + 1))
fi

echo "Checked $count_total curl call(s): $count_with_opts use \$curl_opts."

if [ "$errors" -gt 0 ]; then
  echo "FAILED ($errors error(s))" >&2
  exit 1
fi

echo "PASSED: all curl calls include shared retry/timeout flags."
