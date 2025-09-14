#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
色付きマスク領域の切り取りスクリプト

内容:
- internal/asobi/output内の画像から、色付きマスク領域を検出
- 検出された領域を個別に切り取り
- 元画像と切り取り画像を保存
"""

import cv2
import numpy as np
import os
import json
from pathlib import Path

def detect_colored_regions(image_path, output_dir):
    """
    色付きマスク領域を検出して切り取る
    
    Args:
        image_path (str): 入力画像パス
        output_dir (str): 出力ディレクトリ
    
    Returns:
        list: 切り取った領域の情報
    """
    # 画像を読み込み
    img = cv2.imread(image_path)
    if img is None:
        print(f"画像の読み込みに失敗: {image_path}")
        return []
    
    # ファイル名（拡張子なし）を取得
    base_name = Path(image_path).stem

    # マスク出力用ディレクトリを作成（output_dir/masks）
    mask_output_dir = os.path.join(output_dir, "masks")
    os.makedirs(mask_output_dir, exist_ok=True)

    # HSV色空間に変換
    hsv = cv2.cvtColor(img, cv2.COLOR_BGR2HSV)
    
    # マゼンタ（紫）色範囲の定義（speech_bubble用）
    magenta_lower = np.array([140, 50, 50])
    magenta_upper = np.array([160, 255, 255])

    # シアン（水色）色範囲の定義（square用）
    cyan_lower = np.array([80, 50, 50])
    cyan_upper = np.array([100, 255, 255])
    
    # マスクを作成
    magenta_mask = cv2.inRange(hsv, magenta_lower, magenta_upper)
    cyan_mask = cv2.inRange(hsv, cyan_lower, cyan_upper)
    
    # 全体のマスク
    combined_mask = cv2.bitwise_or(magenta_mask, cyan_mask)
    
    # ノイズ除去
    kernel = np.ones((5, 5), np.uint8)
    combined_mask = cv2.morphologyEx(combined_mask, cv2.MORPH_CLOSE, kernel)
    combined_mask = cv2.morphologyEx(combined_mask, cv2.MORPH_OPEN, kernel)
    
    # 輪郭を検出
    contours, _ = cv2.findContours(combined_mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    
    # 切り取り結果を保存するリスト
    cut_results = []
    
    # 各輪郭について処理
    for i, contour in enumerate(contours):
        # 面積が小さすぎる場合はスキップ
        area = cv2.contourArea(contour)
        if area < 1000:  # 最小面積閾値
            continue
        
        # バウンディングボックスを取得
        x, y, w, h = cv2.boundingRect(contour)
        
        # マージンを追加
        margin = 10
        x_start = max(0, x - margin)
        y_start = max(0, y - margin)
        x_end = min(img.shape[1], x + w + margin)
        y_end = min(img.shape[0], y + h + margin)
        
        # 色の種類を判定
        magenta_pixels = cv2.countNonZero(magenta_mask[y_start:y_end, x_start:x_end])
        cyan_pixels = cv2.countNonZero(cyan_mask[y_start:y_end, x_start:x_end])
        
        if magenta_pixels > cyan_pixels:
            object_type = "speech_bubble"
            color_code = "magenta"
            current_mask = magenta_mask
        else:
            object_type = "square"
            color_code = "cyan"
            current_mask = cyan_mask
        
        # 該当輪郭のマスクを作成
        contour_mask = np.zeros(img.shape[:2], dtype=np.uint8)
        cv2.fillPoly(contour_mask, [contour], 255)
        
        # 色マスクと輪郭マスクの論理積を取る
        precise_mask = cv2.bitwise_and(current_mask, contour_mask)
        
        # 元画像から該当領域を切り取り（矩形領域）
        cropped_region = img[y_start:y_end, x_start:x_end].copy()
        cropped_mask = precise_mask[y_start:y_end, x_start:x_end]
        
        # 色付き部分以外を透明（黒）にする
        # 3チャンネルマスクを作成
        mask_3d = np.stack([cropped_mask, cropped_mask, cropped_mask], axis=-1)

        # マスクが255の部分のみ残し、それ以外は黒にする
        masked_image = np.where(mask_3d == 255, cropped_region, 0)

        # マスク領域に合わせて余白を切り詰め（ぴったりカット）
        ys, xs = np.where(cropped_mask == 255)
        if ys.size > 0 and xs.size > 0:
            y0, y1 = ys.min(), ys.max()
            x0, x1 = xs.min(), xs.max()
            # tight crop を作成
            tight_masked = masked_image[y0:y1+1, x0:x1+1]
            tight_mask = cropped_mask[y0:y1+1, x0:x1+1]
            # 出力座標を更新（元画像座標）
            tight_x_start = x_start + int(x0)
            tight_y_start = y_start + int(y0)
            tight_w = int(x1 - x0 + 1)
            tight_h = int(y1 - y0 + 1)
        else:
            # マスクが空なら矩形のまま保存
            tight_masked = masked_image
            tight_mask = cropped_mask
            tight_x_start = x_start
            tight_y_start = y_start
            tight_w = int(x_end - x_start)
            tight_h = int(y_end - y_start)

        # ファイル名を生成
        output_filename = f"{base_name}_{object_type}_{i:02d}.jpg"
        mask_filename_png = f"{base_name}_{object_type}_{i:02d}_mask.png"

        # 切り取り画像を保存
        output_path = os.path.join(output_dir, output_filename)
        mask_path_png = os.path.join(mask_output_dir, mask_filename_png)

        # tight crop を保存（黒背景）
        cv2.imwrite(output_path, tight_masked)

        # provenance: どの画像のどの座標から切り出したかをわかるように注記画像を作成
        prov_filename = f"{base_name}_{object_type}_{i:02d}_prov.jpg"
        prov_path = os.path.join(output_dir, prov_filename)

        # 注記用の画像を作成（元画像のコピーにテキストを重ねる）
        prov_img = tight_masked.copy()
        prov_text = f"{base_name} @{tight_x_start},{tight_y_start} {tight_w}x{tight_h}"
        font = cv2.FONT_HERSHEY_SIMPLEX
        font_scale = 0.5
        thickness = 1
        # テキストサイズを取得して背景矩形を描く
        (text_w, text_h), _ = cv2.getTextSize(prov_text, font, font_scale, thickness)
        padding = 4
        rect_pt1 = (5, 5)
        rect_pt2 = (5 + text_w + padding*2, 5 + text_h + padding*2)
        # 白背景（視認性確保）
        cv2.rectangle(prov_img, rect_pt1, rect_pt2, (255, 255, 255), -1)
        # テキストを描画（黒）
        text_org = (5 + padding, 5 + text_h + 0)
        cv2.putText(prov_img, prov_text, text_org, font, font_scale, (0, 0, 0), thickness, cv2.LINE_AA)

        # 注記画像を保存
        cv2.imwrite(prov_path, prov_img)

        # マスクを透過PNGとして保存（透明背景 + 白いマスク、アルファはマスク）
        alpha_mask = cv2.GaussianBlur(tight_mask, (5, 5), 0)
        alpha_mask = np.clip(alpha_mask, 0, 255).astype(np.uint8)

        h, w = alpha_mask.shape[:2]
        white_rgb = np.ones((h, w, 3), dtype=np.uint8) * 255
        rgba_mask_image = np.dstack([white_rgb, alpha_mask])

        cv2.imwrite(mask_path_png, rgba_mask_image)

        # 結果情報を記録（JPGマスクは廃止、PNGマスクのみ）
        result_info = {
            "filename": output_filename,
            "prov_filename": prov_filename,
            "mask_filename_png": mask_filename_png,
            "object_type": object_type,
            "color": color_code,
            "area": int(area),
            "bbox": [int(x), int(y), int(w), int(h)],
            "cropped_bbox": [int(tight_x_start), int(tight_y_start), int(tight_w), int(tight_h)],
            "colored_pixels": int(cv2.countNonZero(cropped_mask)),
            "source_image": base_name
        }
        cut_results.append(result_info)

        print(f"切り取り完了: {output_filename} ({object_type}, 面積: {area:.0f}, 色付きピクセル: {cv2.countNonZero(cropped_mask)})")
    
    return cut_results

def process_all_images(input_dir, output_dir):
    """
    指定ディレクトリ内の全画像を処理
    
    Args:
        input_dir (str): 入力ディレクトリ
        output_dir (str): 出力ディレクトリ
    """
    # 出力ディレクトリを作成
    os.makedirs(output_dir, exist_ok=True)
    
    # 処理結果を記録するリスト
    all_results = []
    
    # 対応する画像拡張子
    image_extensions = ['.jpg', '.jpeg', '.png', '.bmp']
    
    # 入力ディレクトリ内のファイルを処理
    for filename in sorted(os.listdir(input_dir)):
        file_path = os.path.join(input_dir, filename)
        
        # 画像ファイルかチェック
        if not any(filename.lower().endswith(ext) for ext in image_extensions):
            continue
        
        print(f"\n処理中: {filename}")
        
        # 矩形切り取り処理を実行
        results = detect_colored_regions(file_path, output_dir)
        
        if results:
            all_results.extend(results)
            print(f"  → 矩形切り取り: {len(results)}個")
        else:
            print(f"  → 色付き領域が検出されませんでした")
    
    # 結果をJSONファイルに保存
    results_file = os.path.join(output_dir, "cut_results.json")
    with open(results_file, 'w', encoding='utf-8') as f:
        json.dump(all_results, f, ensure_ascii=False, indent=2)
    
    print(f"\n=== 処理完了 ===")
    print(f"総切り取り数: {len(all_results)}")
    print(f"結果ファイル: {results_file}")
    
    return all_results

def create_summary_visualization(results, output_dir):
    """
    切り取り結果のサマリー可視化を作成
    
    Args:
        results (list): 切り取り結果
        output_dir (str): 出力ディレクトリ
    """
    if not results:
        return
    
    # オブジェクトタイプ別の統計
    type_counts = {}
    for result in results:
        obj_type = result['object_type']
        type_counts[obj_type] = type_counts.get(obj_type, 0) + 1
    
    # 統計情報をテキストファイルに保存
    summary_file = os.path.join(output_dir, "cut_summary.txt")
    with open(summary_file, 'w', encoding='utf-8') as f:
        f.write("=== 切り取り結果サマリー ===\n\n")
        f.write(f"総切り取り数: {len(results)}\n\n")
        f.write("オブジェクトタイプ別:\n")
        for obj_type, count in type_counts.items():
            f.write(f"  {obj_type}: {count}個\n")
        f.write("\n詳細:\n")
        for i, result in enumerate(results, 1):
            f.write(f"{i:2d}. {result['filename']} "
                   f"({result['object_type']}, 面積: {result['area']})\n")
    
    print(f"サマリーファイル: {summary_file}")

def extract_exact_colored_regions(image_path, output_dir):
    """
    色付き部分のみを正確に抽出（無効化）

    このプロジェクトでは `exact` 出力は不要になったため、
    関数は空の結果を返します。
    """
    print("extract_exact_colored_regions: disabled")
    return []

def main():
    """メイン処理"""
    # パス設定
    input_dir = "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/output"
    output_dir = "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/cut_output"
    
    print("=== 色付きマスク領域切り取りツール ===")
    print(f"入力ディレクトリ: {input_dir}")
    print(f"出力ディレクトリ: {output_dir}")
    
    # 入力ディレクトリの存在確認
    if not os.path.exists(input_dir):
        print(f"エラー: 入力ディレクトリが存在しません: {input_dir}")
        return
    
    # 処理実行
    results = process_all_images(input_dir, output_dir)
    
    # サマリー作成
    create_summary_visualization(results, output_dir)
    
    print(f"\n出力先: {output_dir}")

if __name__ == "__main__":
    main()
