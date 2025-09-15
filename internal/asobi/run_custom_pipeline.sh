#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
PY=python3
GO=go

echo "=== カスタムパイプライン開始 ==="

echo "[1/6] 推論: Bubble.py"
$PY "$REPO_ROOT/internal/asobi/scripts/Bubble.py"

echo "[2/6] 色マスク切り取り: cut.py"
$PY "$REPO_ROOT/internal/asobi/scripts/cut.py"

echo "[3/6] OCR CLI (Go): cmd/api/ocrcli/main.go"
$GO run "$REPO_ROOT/cmd/api/ocrcli/main.go"

echo "[4/6] JSON変換: internal/translate/json_Translated.go"
$GO run "$REPO_ROOT/internal/translate/json_Translated.go"

echo "[5/6] マスクテキスト注釈付け: annotate_masks_with_text.py"
$PY "$REPO_ROOT/internal/asobi/scripts/annotate_masks_with_text.py"

echo "[6/6] マスクテキストを元画像へ貼り付け: paste_back.py"
$PY "$REPO_ROOT/internal/asobi/scripts/paste_back.py"

echo "=== カスタムパイプライン完了 ==="
