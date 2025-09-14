# inference.py
import cv2
import numpy as np
from detectron2.engine import DefaultPredictor
from detectron2.config import get_cfg
import os
from detectron2 import model_zoo

# 1. モデルと設定の読み込み
cfg = get_cfg()
cfg.merge_from_file(model_zoo.get_config_file("COCO-InstanceSegmentation/mask_rcnn_R_50_FPN_3x.yaml"))
cfg.MODEL.ROI_HEADS.NUM_CLASSES = 2  # square と speech_bubble の2クラス
cfg.MODEL.WEIGHTS = os.path.join(cfg.OUTPUT_DIR, "model_final.pth") # 学習済みモデルを指定
cfg.MODEL.ROI_HEADS.SCORE_THRESH_TEST = 0.5  # 信頼度の閾値を学習時と一致させる
predictor = DefaultPredictor(cfg)

# 2. 新しい画像の読み込みと推論
input_dir = "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/こわいやさん[第1話]"
output_dir = "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/output"
os.makedirs(output_dir, exist_ok=True)

for filename in os.listdir(input_dir):
    if not filename.lower().endswith(('.png', '.jpg', '.jpeg')):
        continue

    image_path = os.path.join(input_dir, filename)
    im = cv2.imread(image_path)
    if im is None:
        print(f"Warning: Could not read image {image_path}. Skipping.")
        continue

    outputs = predictor(im)

    # 3. マスクの抽出と切り抜き処理
    instances = outputs["instances"].to("cpu")
    masks = instances.pred_masks.numpy()
    scores = instances.scores.numpy()
    classes = instances.pred_classes.numpy()

    # クラス名を定義（学習時と一致させる）
    class_names = {0: "square", 1: "speech_bubble"}

    # 検出された全てのオブジェクトに対してループ
    for i, (mask, score, class_id) in enumerate(zip(masks, scores, classes)):
        class_name = class_names.get(class_id, f"unknown_{class_id}")
        print(f"検出: {class_name} (信頼度: {score:.3f})")
        
        # speech_bubbleのみ処理対象とする場合
        if class_id == 1:  # speech_bubbleのクラスID
            # マスクは (高さ, 幅) のTrue/False配列
            # このマスクを使って処理を行う
            
            # 例1: 吹き出し部分を半透明のマゼンタ（紫）で塗りつぶす
            overlay = im.copy()
            overlay[mask] = [255, 0, 255]  # BGRなのでマゼンタ（紫）
            alpha = 0.5  # 透明度
            im = cv2.addWeighted(overlay, alpha, im, 1 - alpha, 0)
            
            # 例2: 吹き出し部分だけを切り出す（黒背景）
            # (高さ, 幅, 3チャンネル) に変換
            mask_3d = np.stack([mask, mask, mask], axis=-1)
            # マスクされた部分だけを抽出
            cropped_bubble = np.where(mask_3d, im, 0)
            # cv2.imwrite(f"bubble_{i}.png", cropped_bubble)
        
        elif class_id == 0:  # squareの場合
            # squareはシアン（水色）で表示
            overlay = im.copy()
            overlay[mask] = [255, 255, 0]  # BGRなのでシアン（水色）
            alpha = 0.3  # より薄い透明度
            im = cv2.addWeighted(overlay, alpha, im, 1 - alpha, 0)

    # 結果の表示・保存
    output_path = os.path.join(output_dir, filename)
    cv2.imwrite(output_path, im)
    print(f"Processed {filename} and saved to {output_path}")