#!/usr/bin/env bash
set -euo pipefail

launcher_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
backend_dir="$launcher_dir/../backend"
config_file="$backend_dir/.env"

if [[ ! -f "$config_file" ]]; then
    echo "Create backend/.env using backend/.env.example, then set your model and vision projector paths." >&2
    exit 1
fi

# Resolve both configuration and relative model paths from backend/, not the caller's directory.
cd -- "$backend_dir"
source "$config_file"

if [[ -z "${RECEIPT_LLM_MODEL_PATH:-}" || -z "${RECEIPT_LLM_MMPROJ_PATH:-}" ]]; then
    echo "Set RECEIPT_LLM_MODEL_PATH and RECEIPT_LLM_MMPROJ_PATH in backend/.env first." >&2
    exit 1
fi

if [[ ! -f "$RECEIPT_LLM_MODEL_PATH" || ! -r "$RECEIPT_LLM_MODEL_PATH" ]]; then
    echo "Cannot read the model GGUF. Check RECEIPT_LLM_MODEL_PATH in backend/.env." >&2
    exit 1
fi

if [[ ! -f "$RECEIPT_LLM_MMPROJ_PATH" || ! -r "$RECEIPT_LLM_MMPROJ_PATH" ]]; then
    echo "Cannot read the vision projector GGUF. Check RECEIPT_LLM_MMPROJ_PATH in backend/.env." >&2
    exit 1
fi

if ! command -v llama-server >/dev/null 2>&1; then
    echo "llama-server was not found on PATH. Install a llama.cpp build with the server executable." >&2
    exit 1
fi

exec llama-server \
    -m "$RECEIPT_LLM_MODEL_PATH" \
    --mmproj "$RECEIPT_LLM_MMPROJ_PATH" \
    -ngl "${RECEIPT_LLM_GPU_LAYERS:-99}" \
    -c "${RECEIPT_LLM_CONTEXT:-40960}" \
    -fa on --jinja \
    --chat-template-kwargs '{"enable_thinking":false}' \
    --alias "${RECEIPT_LLM_MODEL:-receipts}" \
    --host 127.0.0.1 \
    --port "${RECEIPT_LLM_PORT:-8081}"
