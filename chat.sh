#!/usr/bin/env bash
set -e

MODEL="$1"
THINKING="${2:-off}"
CONTEXT="${3:-40960}"
IMAGE="${4:-}"

if [[ ! -f "$MODEL" ]]; then
    echo "Usage: $0 <model.gguf> [off|on|auto] [context] [image]"
    exit 1
fi

if [[ ! "$THINKING" =~ ^(off|on|auto)$ ]]; then
    echo "Invalid thinking mode: $THINKING"
    exit 1
fi

ARGS=(
    -m "$MODEL"
    -ngl 99
    -c "$CONTEXT"
    -fa on
    --jinja
)

# Configure thinking mode
case "$THINKING" in
    off)
        ARGS+=(
            --chat-template-kwargs '{"enable_thinking":false}'
        )
        ;;
    on)
        ARGS+=(
            --chat-template-kwargs '{"enable_thinking":true}'
        )
        ;;
    auto)
        # Use model's default behavior
        ;;
esac

# Image description mode
if [[ -n "$IMAGE" ]]; then
    MMPROJ="$(dirname "$MODEL")/mmproj-F16.gguf"

    if [[ ! -f "$MMPROJ" ]]; then
        echo "Vision projector not found: $MMPROJ"
        exit 1
    fi

    if [[ ! -f "$IMAGE" ]]; then
        echo "Image not found: $IMAGE"
        exit 1
    fi

    ARGS+=(
        --mmproj "$MMPROJ"
        --image "$IMAGE"
        -p "Describe this image in detail."
    )
fi

exec llama-cli "${ARGS[@]}"
