#!/usr/bin/env bash
# Install a portable Python + pymupdf/reportlab under /app/.venv-pdf (fallback: ./.venv-pdf).
set -euo pipefail

echo "[pdf-runtime] start pwd=$(pwd)"

if [[ -d /app && -w /app ]]; then
  VENV_DIR="/app/.venv-pdf"
else
  VENV_DIR="$(pwd)/.venv-pdf"
fi
mkdir -p "$(dirname "$VENV_DIR")" /tmp

# Best-effort system packages (ignored if apt unavailable).
if command -v apt-get >/dev/null 2>&1; then
  apt-get update -qq || true
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
    python3 python3-pip python3-venv poppler-utils ca-certificates curl \
    || true
fi

PYTHON_BIN=""
for cand in python3 python /usr/bin/python3 /usr/local/bin/python3; do
  if command -v "$cand" >/dev/null 2>&1; then
    PYTHON_BIN="$(command -v "$cand")"
    break
  fi
done

if [[ -z "$PYTHON_BIN" ]]; then
  echo "[pdf-runtime] installing uv + CPython (no system python)"
  curl -LsSf https://astral.sh/uv/install.sh | sh
  export PATH="${HOME}/.cargo/bin:${HOME}/.local/bin:${PATH}"
  uv python install 3.12
  PYTHON_BIN="$(uv python find 3.12)"
fi

echo "[pdf-runtime] python=$PYTHON_BIN venv=$VENV_DIR"
"$PYTHON_BIN" -m venv "$VENV_DIR"
"$VENV_DIR/bin/pip" install --upgrade pip
"$VENV_DIR/bin/pip" install --no-cache-dir pypdf reportlab pymupdf
"$VENV_DIR/bin/python" -c 'import pymupdf; print("[pdf-runtime] pymupdf", pymupdf.version)'

# Also keep a copy in the build workspace so later build steps can find it.
if [[ "$VENV_DIR" == "/app/.venv-pdf" && "$(pwd)" != "/app" ]]; then
  rm -rf "$(pwd)/.venv-pdf"
  cp -a "$VENV_DIR" "$(pwd)/.venv-pdf" || true
fi

echo "[pdf-runtime] done"
