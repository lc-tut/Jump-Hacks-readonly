#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
translated_result.json の text を対応する mask 画像に描画して保存するスクリプト

- 入力: translated_result.json（リポジトリルート）
- マスクディレクトリ: internal/asobi/cut_output/masks
- 出力: 元のマスク画像は変更せず、同ディレクトリに "*_mask_text.png" を作成

使い方:
    python3 internal/asobi/scripts/annotate_masks_with_text.py

"""

import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
import textwrap
import os
import sys

REPO_ROOT = Path(__file__).resolve().parents[3]
TRANSLATED_JSON = REPO_ROOT / "translated_result.json"
MASKS_DIR = REPO_ROOT / "internal" / "asobi" / "cut_output" / "masks"
FONT_FILE = REPO_ROOT / "font.ttf"
TEXT_OUTPUT_DIR = REPO_ROOT / "internal" / "asobi" / "textim"


def load_translations(json_path):
    if not json_path.exists():
        print(f"translated_result.json が見つかりません: {json_path}")
        return []
    with open(json_path, 'r', encoding='utf-8') as f:
        return json.load(f)


def find_mask_for_source(source_path):
    # source: .../cut_output/0001_speech_bubble_02.jpg
    base = Path(source_path).stem  # 0001_speech_bubble_02
    # mask filename pattern used by切り取りスクリプト: <base>_mask.png
    mask_name = f"{base}_mask.png"
    return MASKS_DIR / mask_name


def get_font(size):
    if FONT_FILE.exists():
        try:
            return ImageFont.truetype(str(FONT_FILE), size=size)
        except Exception:
            pass
    return ImageFont.load_default()


def wrap_text(text, draw, font, max_width):
    # textwrap で幅を決めるために文字単位の目安を作る
    sample = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZあいうえお'
    widths = []
    for c in sample:
        try:
            bbox = draw.textbbox((0, 0), c, font=font)
            widths.append(bbox[2] - bbox[0])
        except Exception:
            try:
                w = draw.textsize(c, font=font)[0]
                widths.append(w)
            except Exception:
                widths.append(7)

    if widths:
        avg_char_width = sum(widths) / len(widths)
    else:
        avg_char_width = 7

    if avg_char_width <= 0:
        avg_char_width = 7

    max_chars = max(10, int(max_width / avg_char_width))
    wrapped = textwrap.fill(text, width=max_chars)
    return wrapped


def annotate_mask(mask_path: Path, text: str):
    if not mask_path.exists():
        print(f"マスクが見つかりません: {mask_path}")
        return False

    try:
        img = Image.open(mask_path).convert("RGBA")
    except Exception as e:
        print(f"画像を開けませんでした: {mask_path} -> {e}")
        return False

    w, h = img.size

    # アルファチャンネル（マスク）を取得
    alpha = img.split()[3]
    bbox = alpha.getbbox()
    if bbox is None:
        print(f"マスクが空です: {mask_path}")
        return False

    box_x0, box_y0, box_x1, box_y1 = bbox
    box_w = box_x1 - box_x0
    box_h = box_y1 - box_y0

    # フォントサイズは画像サイズに応じて決定
    font_size = max(12, int(min(w, h) * 0.04))
    font = get_font(font_size)

    # テキストをラップ（マスク領域幅に合わせる）
    padding = 8
    max_text_width = max(10, box_w - padding * 2)

    # テキストを領域幅に合わせて折り返すため、一時的な ImageDraw を作成
    tmp = Image.new('RGBA', (box_w, box_h), (255,255,255,0))
    tmp_draw = ImageDraw.Draw(tmp)
    wrapped = wrap_text(text, tmp_draw, font, max_text_width)
    lines = wrapped.split('\n')

    # 行高さを計算
    try:
        bbox_line = tmp_draw.textbbox((0, 0), 'Ay', font=font)
        line_height = (bbox_line[3] - bbox_line[1]) + 4
    except Exception:
        try:
            _w, _h = tmp_draw.textsize('Ay', font=font)
            line_height = _h + 4
        except Exception:
            line_height = getattr(font, 'size', 16) + 4

    text_block_height = line_height * len(lines)

    # マスク領域を白背景（不透明）にする画像を作成
    white_bg = Image.new('RGBA', img.size, (255,255,255,0))
    white_full = Image.new('RGBA', img.size, (255,255,255,255))
    # alpha を使って白のみ表示する（マスクの形状を保持）
    white_full.putalpha(alpha)

    # テキストオーバーレイ（透明背景）を作成し、マスク領域の矩形内にテキストを中央配置
    text_overlay = Image.new('RGBA', img.size, (255,255,255,0))
    todraw = ImageDraw.Draw(text_overlay)

    # テキスト描画開始位置（矩形内で中央揃え）
    start_y = box_y0 + max(0, (box_h - text_block_height) // 2)
    for line in lines:
        try:
            tb = todraw.textbbox((0,0), line, font=font)
            tw = tb[2] - tb[0]
        except Exception:
            tw = todraw.textsize(line, font=font)[0]
        x = box_x0 + max(0, (box_w - tw) // 2)
        todraw.text((x, start_y), line, font=font, fill=(0,0,0,255))
        start_y += line_height

    # 白背景（マスク形状）とテキストを合成
    combined = Image.alpha_composite(white_full, text_overlay)

    # 出力保存
    out_name = mask_path.stem + "_text.png"
    TEXT_OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    out_path = TEXT_OUTPUT_DIR / out_name
    combined.save(out_path)
    print(f"書き込み: {out_path}")
    return True


def main():
    translations = load_translations(TRANSLATED_JSON)
    if not translations:
        print("翻訳結果が空です。")
        return

    if not MASKS_DIR.exists():
        print(f"マスクディレクトリが存在しません: {MASKS_DIR}")
        return

    count = 0
    for item in translations:
        src = item.get('source')
        text = item.get('text', '')
        if not src or not text:
            continue
        mask_path = find_mask_for_source(src)
        ok = annotate_mask(mask_path, text)
        if ok:
            count += 1

    print(f"完了: {count} 件のマスクに注釈を追加しました")


if __name__ == '__main__':
    main()
