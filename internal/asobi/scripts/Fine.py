# train.py
import detectron2
from detectron2.utils.logger import setup_logger
setup_logger()

from detectron2.data.datasets import register_coco_instances
from detectron2.engine import DefaultTrainer
from detectron2.config import get_cfg
import os
from detectron2 import model_zoo

# 1. データセットの登録
# "my_dataset_train" という名前でデータセットを登録
register_coco_instances("my_dataset_train", {}, 
                        "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/asobi_coco/annotations.json", 
                        "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/asobi_coco/")
register_coco_instances("my_dataset_val", {}, 
                        "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/asobi_coco/annotations.json", 
                        "/home/luy869/works/hackathon/Jump-Hacks-readonly/internal/asobi/asobi_coco/")

# データセットの統計情報を確認
from detectron2.data import DatasetCatalog, MetadataCatalog
dataset_dicts = DatasetCatalog.get("my_dataset_train")
print(f"データセット内の画像数: {len(dataset_dicts)}")

# クラス分布を確認
class_counts = {}
for d in dataset_dicts:
    for ann in d["annotations"]:
        category_id = ann["category_id"]
        if category_id not in class_counts:
            class_counts[category_id] = 0
        class_counts[category_id] += 1

print("クラス分布:")
metadata = MetadataCatalog.get("my_dataset_train")
for cat_id, count in class_counts.items():
    print(f"  クラスID {cat_id}: {count}個")

# 2. 設定（コンフィグ）の準備
cfg = get_cfg()
# 事前学習済みのMask R-CNNモデルをベースにする
cfg.merge_from_file(model_zoo.get_config_file("COCO-InstanceSegmentation/mask_rcnn_R_50_FPN_3x.yaml"))
cfg.MODEL.WEIGHTS = model_zoo.get_checkpoint_url("COCO-InstanceSegmentation/mask_rcnn_R_50_FPN_3x.yaml")

# 3. 設定のカスタマイズ
cfg.DATASETS.TRAIN = ("my_dataset_train",)
cfg.DATASETS.TEST = ()  # 検証データは使わない場合
cfg.DATALOADER.NUM_WORKERS = 8   # ワーカー数を減らして安定化
cfg.SOLVER.IMS_PER_BATCH = 4  # バッチサイズを小さくして安定した学習
cfg.SOLVER.BASE_LR = 0.00005    # 学習率を下げて安定した学習
cfg.SOLVER.MAX_ITER = 1000     # 学習イテレーション数を増やす
cfg.SOLVER.STEPS = (700, 900)  # 学習率を段階的に下げる
cfg.SOLVER.GAMMA = 0.1         # 学習率の減衰率
cfg.SOLVER.WARMUP_ITERS = 100  # ウォームアップ期間
cfg.SOLVER.WARMUP_FACTOR = 1.0 / 100
cfg.MODEL.ROI_HEADS.BATCH_SIZE_PER_IMAGE = 64  # ROIのバッチサイズを小さく
cfg.MODEL.ROI_HEADS.NUM_CLASSES = 2  # square と speech_bubble の2クラス
cfg.MODEL.ROI_HEADS.SCORE_THRESH_TEST = 0.5   # 推論時の閾値を設定

# アンカーサイズを調整（オブジェクトサイズに合わせて）
cfg.MODEL.ANCHOR_GENERATOR.SIZES = [[32], [64], [128], [256], [512]]
cfg.MODEL.ANCHOR_GENERATOR.ASPECT_RATIOS = [[0.5, 1.0, 2.0]]

# Data Augmentation を有効化
cfg.INPUT.MIN_SIZE_TRAIN = (640, 672, 704, 736, 768, 800)
cfg.INPUT.MAX_SIZE_TRAIN = 1333
cfg.INPUT.MIN_SIZE_TEST = 800
cfg.INPUT.MAX_SIZE_TEST = 1333

# 4. 学習の開始
os.makedirs(cfg.OUTPUT_DIR, exist_ok=True)

# TensorBoard用のログ設定
from detectron2.utils.events import TensorboardXWriter
from detectron2.engine import HookBase
import detectron2.utils.comm as comm

class ValidationLoss(HookBase):
    def __init__(self):
        super().__init__()

    def after_step(self):
        if self.trainer.iter % 100 == 0:  # 100回ごとにログ出力
            print(f"Iteration {self.trainer.iter}: Loss = {self.trainer.storage.latest()['total_loss']}")

trainer = DefaultTrainer(cfg) 
trainer.register_hooks([ValidationLoss()])
trainer.resume_or_load(resume=False)

print("学習を開始します...")
print(f"設定:")
print(f"  バッチサイズ: {cfg.SOLVER.IMS_PER_BATCH}")
print(f"  学習率: {cfg.SOLVER.BASE_LR}")
print(f"  最大イテレーション: {cfg.SOLVER.MAX_ITER}")
print(f"  クラス数: {cfg.MODEL.ROI_HEADS.NUM_CLASSES}")

trainer.train()

# 5. 推論の実行
print("学習完了！推論を実行します...")

from detectron2.engine import DefaultPredictor
from detectron2.utils.visualizer import Visualizer, ColorMode
import cv2
import matplotlib.pyplot as plt

# 推論用設定
cfg.MODEL.WEIGHTS = os.path.join(cfg.OUTPUT_DIR, "model_final.pth")
cfg.MODEL.ROI_HEADS.SCORE_THRESH_TEST = 0.5
predictor = DefaultPredictor(cfg)

# テスト画像で推論
dataset_dicts = DatasetCatalog.get("my_dataset_train")
metadata = MetadataCatalog.get("my_dataset_train")

# 最初の3つの画像で推論をテスト
for i, d in enumerate(dataset_dicts[:3]):
    im = cv2.imread(d["file_name"])
    outputs = predictor(im)
    
    # 予測結果の可視化
    v = Visualizer(im[:, :, ::-1], metadata=metadata, scale=0.8, instance_mode=ColorMode.IMAGE_BW)
    out = v.draw_instance_predictions(outputs["instances"].to("cpu"))
    
    # 結果を保存
    result_image = out.get_image()[:, :, ::-1]
    cv2.imwrite(f"result_{i}.jpg", result_image)
    print(f"結果画像を保存しました: result_{i}.jpg")
    
    # 検出結果の統計
    instances = outputs["instances"]
    print(f"画像 {i}: {len(instances)} 個のオブジェクトを検出")
    for j, (score, class_id) in enumerate(zip(instances.scores, instances.pred_classes)):
        class_name = metadata.thing_classes[class_id] if hasattr(metadata, 'thing_classes') else f"class_{class_id}"
        print(f"  オブジェクト {j}: {class_name} (信頼度: {score:.3f})")

print("推論完了！")