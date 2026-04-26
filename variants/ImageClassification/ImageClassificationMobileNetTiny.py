from transformers import pipeline

# DeiT-Tiny for image classification — minimum-footprint variant.
# ~5.7M parameters, 72.2% ImageNet Top-1 accuracy.
# Designed for extremely resource-constrained environments; minimal energy
# cost per inference.  (error_estimate = 1 - 0.722 ≈ 0.278)
_classifier = pipeline(
    "image-classification",
    model="facebook/deit-tiny-patch16-224",
    device=-1,
)


def handler(params, context):
    image_base64 = params["image_base64"]
    result = _classifier(image_base64)[0]
    return {
        "label": result["label"],
        "confidence": float(result["score"]),
    }
