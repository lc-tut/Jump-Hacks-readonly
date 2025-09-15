#!/bin/bash

# いずれかのコマンドが失敗した場合、スクリプトを直ちに停止する
set -e

echo "===== AI学習・推論パイプラインを開始します ====="

# --- ステップ0: データ整合性チェック ---
echo ""
echo "[ステップ0/5] データの整合性をチェックします..."

# 対応する画像ファイルが存在しないJSONファイルを確認
cd /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi
missing_images=()
for json_file in json/*.json; do
    base_name=$(basename "$json_file" .json)
    if [ ! -f "asobi_coco/JPEGImages/${base_name}.jpg" ] && [ ! -f "json/../JPEGImages/${base_name}.jpg" ]; then
        # 画像ファイルが見つからない場合、JSONファイルを一時的に移動
        echo "警告: ${base_name}.jpg が見つかりません。${json_file} を一時的に無効化します。"
        mv "$json_file" "${json_file}.backup"
        missing_images+=("$base_name")
    fi
done

if [ ${#missing_images[@]} -eq 0 ]; then
    echo "✓ 全てのJSONファイルに対応する画像ファイルが存在します。"
else
    echo "⚠ 以下のファイルは画像が不足しているため学習から除外されました: ${missing_images[*]}"
fi

# --- ステップ1: 古いデータのクリーンアップ ---
echo ""
echo "[ステップ1/5] 古い生成データをクリーンアップします..."
rm -rf /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/asobi_coco
rm -rf /home/luy869/works/hackathon/Jump-Hacks-readonly/output
echo "クリーンアップ完了。"


# --- ステップ2: LabelMeからCOCO形式へ変換 ---
echo ""
echo "[ステップ2/5] 新しいアノテーションデータ（JSON）をCOCO形式に変換します..."
python /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/scripts/labelme2coco.py \
/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/json \
/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/asobi_coco \
--labels /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/json/labels.txt
echo "データ変換完了。"


# --- ステップ3: モデルの学習 ---
echo ""
echo "[ステップ3/5] 新しいデータセットでAIモデルを学習します..."
python /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/scripts/Fine.py
echo "モデルの学習完了。"


# --- ステップ4: 推論の実行 ---
echo ""
echo "[ステップ4/5] 学習済みモデルを使って推論を実行します..."
python /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/scripts/Bubble.py
echo "推論完了。'output_image.jpg' が生成されました。"

# --- ステップ5: バックアップファイルの復元 ---
echo ""
echo "[ステップ5/5] バックアップファイルを復元します..."
cd /home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi
for backup_file in json/*.json.backup; do
    if [ -f "$backup_file" ]; then
        original_file="${backup_file%.backup}"
        mv "$backup_file" "$original_file"
        echo "復元: $(basename "$original_file")"
    fi
done
echo "バックアップファイルの復元完了。"

echo ""
echo "===== パイプラインの全工程が正常に完了しました！ ====="
