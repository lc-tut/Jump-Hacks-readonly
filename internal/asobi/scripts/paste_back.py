#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
mask_text2 内の画像を元の位置に貼り付けて、出力先に保存するスクリプト
参照: internal/asobi/cut_output/cut_results.json
出力先: internal/asobi/koko
"""
import os
import cv2
import json
from pathlib import Path
import glob
import numpy as np

BASE = Path(__file__).resolve().parents[1]  # internal/asobi
CUT_OUTPUT = BASE / 'cut_output'
# マスクテキストのソースディレクトリを `cut_output/mask_text2` から `internal/asobi/textim` に変更
MASK_TEXT_DIR = BASE / 'textim'
RESULTS_JSON = CUT_OUTPUT / 'cut_results.json'
ORIG_DIR = BASE / 'output'
OUT_DIR = BASE / 'koko'

IMAGE_EXTS = ['.jpg', '.jpeg', '.png', '.bmp']


def find_source_image(base_name):
    # source image may have any extension in ORIG_DIR
    for ext in IMAGE_EXTS:
        candidate = ORIG_DIR / (base_name + ext)
        if candidate.exists():
            return str(candidate)
    # try glob
    matches = list(ORIG_DIR.glob(base_name + '.*'))
    if matches:
        return str(matches[0])
    return None


def load_image_rgba(path):
    # load with alpha if present
    img = cv2.imread(str(path), cv2.IMREAD_UNCHANGED)
    if img is None:
        return None
    # if grayscale, convert to BGR
    if img.ndim == 2:
        img = cv2.cvtColor(img, cv2.COLOR_GRAY2BGR)
    # if BGR (3ch) -> add full alpha
    if img.shape[2] == 3:
        b,g,r = cv2.split(img)
        alpha = np.ones_like(b) * 255
        img = cv2.merge([b,g,r,alpha])
    return img


def alpha_composite(base_bgr, overlay_rgba, x, y, w, h):
    H, W = base_bgr.shape[:2]
    # clip region
    if x >= W or y >= H:
        print(f"warning: overlay out of bounds: x={x},y={y} base={W}x{H}")
        return base_bgr
    ow = min(w, W - x)
    oh = min(h, H - y)
    if ow <= 0 or oh <= 0:
        return base_bgr

    # resize overlay if necessary
    if overlay_rgba.shape[1] != ow or overlay_rgba.shape[0] != oh:
        overlay_rgba = cv2.resize(overlay_rgba, (ow, oh), interpolation=cv2.INTER_AREA)

    overlay_rgb = overlay_rgba[..., :3].astype(np.float32) / 255.0
    alpha = overlay_rgba[..., 3].astype(np.float32) / 255.0
    alpha = np.expand_dims(alpha, axis=2)

    roi = base_bgr[y:y+oh, x:x+ow].astype(np.float32) / 255.0
    comp = overlay_rgb * alpha + roi * (1.0 - alpha)
    base_bgr[y:y+oh, x:x+ow] = np.clip(comp * 255.0, 0, 255).astype(np.uint8)
    return base_bgr


def main():
    os.makedirs(OUT_DIR, exist_ok=True)

    if not RESULTS_JSON.exists():
        print(f"結果JSONが見つかりません: {RESULTS_JSON}")
        return
    if not MASK_TEXT_DIR.exists():
        print(f"mask_text2ディレクトリが見つかりません: {MASK_TEXT_DIR}")
        return

    with open(RESULTS_JSON, 'r', encoding='utf-8') as f:
        results = json.load(f)

    # build lookup by stems
    lookup = {}
    for r in results:
        # keys to match: filename (stem), mask_filename_png (stem), prov_filename (stem) if present
        for key in ('filename', 'mask_filename_png', 'prov_filename'):
            if key in r and r[key]:
                stem = Path(r[key]).stem
                lookup[stem] = r

    # group overlays by source_image
    groups = {}
    for file in sorted(MASK_TEXT_DIR.iterdir()):
        if not file.is_file():
            continue
        stem = file.stem
        if stem in lookup:
            r = lookup[stem]
            src = r.get('source_image') or Path(r.get('filename')).stem
            groups.setdefault(src, []).append((file, r))
        else:
            # try fuzzy match: if stem contains "_" and last part is like 00, try prefix
            parts = stem.split('_')
            if len(parts) >= 2:
                prefix = '_'.join(parts[:-1])
                if prefix in lookup:
                    r = lookup[prefix]
                    src = r.get('source_image') or Path(r.get('filename')).stem
                    groups.setdefault(src, []).append((file, r))
                else:
                    print(f"未マッチのファイル: {file.name}")
            else:
                print(f"未マッチのファイル: {file.name}")

    # process each source image, paste overlays
    for src_base, overlays in groups.items():
        src_path = find_source_image(src_base)
        if not src_path:
            print(f"元画像が見つかりません: {src_base} (検索パス: {ORIG_DIR})")
            continue
        src_img = cv2.imread(src_path, cv2.IMREAD_COLOR)
        if src_img is None:
            print(f"元画像の読み込み失敗: {src_path}")
            continue

        canvas = src_img.copy()

        for overlay_file, r in overlays:
            # load overlay with alpha if present
            over = load_image_rgba(str(overlay_file))
            if over is None:
                print(f"オーバーレイ読み込み失敗: {overlay_file}")
                continue

            # target bbox from result
            x, y, w, h = r.get('cropped_bbox', [0,0,over.shape[1], over.shape[0]])
            x = int(x); y = int(y); w = int(w); h = int(h)

            # if overlay larger/smaller, resizing handled in alpha_composite
            canvas = alpha_composite(canvas, over, x, y, w, h)
            print(f"貼り付け: {overlay_file.name} -> {Path(src_path).name} @ ({x},{y}) {w}x{h}")

        out_name = f"{src_base}_pasted.png"
        out_path = OUT_DIR / out_name
        cv2.imwrite(str(out_path), canvas)
        print(f"保存: {out_path}")

    print("処理完了")

if __name__ == '__main__':
    main()
