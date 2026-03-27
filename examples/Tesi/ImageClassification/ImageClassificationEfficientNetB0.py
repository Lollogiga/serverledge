from transformers import pipeline

# EfficientNet-B0 for image classification.
# ~5.3M parameters, 77.1% ImageNet Top-1 accuracy.
# Significantly lighter than ViT-base (86M params, 81.8%) while retaining
# competitive accuracy.  (error_estimate = 1 - 0.771 ≈ 0.229)
_classifier = pipeline(
    "image-classification",
    model="google/efficientnet_b0",
    device=-1,
)


def handler(params, context):
    image_base64 = params["image_base64"]
    result = _classifier(image_base64)[0]
    return {
        "label": result["label"],
        "confidence": float(result["score"]),
    }
