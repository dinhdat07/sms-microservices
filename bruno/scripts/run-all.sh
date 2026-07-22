#!/bin/bash
# Run all Bruno API smoke tests
set -e
ENV="${1:-local}"
COLLECTION_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Bruno API Smoke Tests - SMS ==="
echo "Environment: $ENV"
cd "$COLLECTION_DIR"

FOLDERS=("health" "auth" "servers" "reporting" "authorization")
FAILED=()
for folder in "${FOLDERS[@]}"; do
    echo "--- $folder ---"
    if bru run "$folder" --env "$ENV"; then
        echo "  PASSED"
    else
        FAILED+=("$folder")
        echo "  FAILED"
    fi
done

echo ""
if [ ${#FAILED[@]} -eq 0 ]; then
    echo "ALL TESTS PASSED"
    exit 0
else
    echo "FAILED: ${FAILED[*]}"
    exit 1
fi
