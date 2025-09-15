#!/usr/bin/env python3
"""
Paste images from internal/asobi/textim onto corresponding images in internal/asobi/output
using internal/asobi/cut_output/cut_results.json as mapping, and write results to internal/asobi/koko.

Usage:
    python3 internal/asobi/paste_textim_to_output.py
"""
import json
import os
from collections import defaultdict
from PIL import Image

BASE_DIR = os.path.dirname(os.path.abspath(__file__))
CUT_JSON = os.path.join(BASE_DIR, 'cut_output', 'cut_results.json')
TEXTIM_DIR = os.path.join(BASE_DIR, 'textim')
OUTPUT_DIR = os.path.join(BASE_DIR, 'output')
KOKO_DIR = os.path.join(BASE_DIR, 'koko')
os.makedirs(KOKO_DIR, exist_ok=True)


def find_output_image(source_base):
    # try common extensions
    for ext in ('.jpg', '.jpeg', '.png'):
        p = os.path.join(OUTPUT_DIR, source_base + ext)
        if os.path.exists(p):
            return p
    return None


def find_textim_image(cut_filename):
    # cut_filename examples: 0001_speech_bubble_00.jpg
    base = cut_filename.rsplit('.', 1)[0]
    # common textim naming in repo: <base>_mask_text.png
    candidates = [base + '_mask_text.png', base + '_mask.png', base + '_text.png', base + '.png']
    for c in candidates:
        p = os.path.join(TEXTIM_DIR, c)
        if os.path.exists(p):
            return p
    return None


def main():
    with open(CUT_JSON, 'r') as f:
        cut_entries = json.load(f)

    by_source = defaultdict(list)
    for e in cut_entries:
        src = e.get('source_image')
        if not src:
            continue
        by_source[src].append(e)

    for source, entries in sorted(by_source.items()):
        out_path = find_output_image(source)
        if out_path is None:
            print(f"skip {source}: output image not found")
            continue

        base_img = Image.open(out_path).convert('RGBA')
        for e in entries:
            cut_fname = e.get('filename')
            bbox = e.get('bbox')  # [x, y, w, h]
            if not (cut_fname and bbox):
                continue

            textim_path = find_textim_image(cut_fname)
            if textim_path is None:
                print(f"textim not found for {cut_fname} (source {source})")
                continue

            try:
                txt = Image.open(textim_path).convert('RGBA')
            except Exception as ex:
                print(f"failed open textim {textim_path}: {ex}")
                continue

            x, y, w, h = bbox
            # Resize textim to bbox size if sizes differ
            if txt.size != (w, h):
                txt = txt.resize((w, h), resample=Image.LANCZOS)

            # Paste using alpha channel
            base_img.paste(txt, (x, y), txt)
            print(f"pasted {os.path.basename(textim_path)} onto {os.path.basename(out_path)} at {(x,y)}")

        save_path = os.path.join(KOKO_DIR, f"{source}_pasted.png")
        base_img.save(save_path)
        print(f"saved {save_path}")


if __name__ == '__main__':
    main()
