from transformers import pipeline

# MobileNetV2 (width multiplier 0.35) — minimum-footprint variant.
# ~1.7M parameters, 60.3% ImageNet Top-1 accuracy.
# Designed for extremely resource-constrained environments; minimal energy
# cost per inference.  (error_estimate = 1 - 0.603 ≈ 0.397)
_classifier = pipeline(
    "image-classification",
    model="google/mobilenet_v2_0.35_224",
    device=-1,
)


def handler(params, context):
    image_base64 = params["image_base64"]
    result = _classifier(image_base64)[0]
    return {
        "label": result["label"],
        "confidence": float(result["score"]),
    }
