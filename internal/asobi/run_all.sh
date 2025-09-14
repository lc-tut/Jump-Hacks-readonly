#!/usr/bin/env bash
set -euo pipefail

# このスクリプトは以下を順に実行します:
# 1) 色付き領域の切り出し: scripts/cut.py
# 2) 切り出し結果を元画像に貼り付け: scripts/paste_back.py
# 3) （任意）学習処理: scripts/Fine.py

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="$ROOT_DIR/scripts"

PYTHON=${PYTHON:-python3}

echo "=== run_all: START ==="

# 1) cut.py
if [ -f "$SCRIPTS_DIR/cut.py" ]; then
  echo "[1/3] running cut.py"
  "$PYTHON" "$SCRIPTS_DIR/cut.py"
else
  echo "warn: $SCRIPTS_DIR/cut.py が見つかりません。スキップします。"
fi

# 2) paste_back.py
if [ -f "$SCRIPTS_DIR/paste_back.py" ]; then
  echo "[2/3] running paste_back.py"
  "$PYTHON" "$SCRIPTS_DIR/paste_back.py"
else
  echo "warn: $SCRIPTS_DIR/paste_back.py が見つかりません。スキップします。"
fi

# 3) Fine.py (学習) - オプション
# 実行するには引数に --train を付ける
if [ "${1-}" = "--train" ] || [ "${1-}" = "train" ]; then
  if [ -f "$SCRIPTS_DIR/Fine.py" ]; then
    echo "[3/3] running Fine.py (training)"
    echo "注意: 学習は計算資源を多く消費します。GPU環境で実行してください。"
    "$PYTHON" "$SCRIPTS_DIR/Fine.py"
  else
    echo "warn: $SCRIPTS_DIR/Fine.py が見つかりません。スキップします。"
  fi
else
  echo "[3/3] Fine.py (training) はスキップされました。学習を実行するには: $0 --train"
fi

echo "=== run_all: DONE ==="
