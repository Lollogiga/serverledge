from transformers import pipeline

classifier = pipeline(
    "image-classification",
    model="google/mobilenet_v2_1.0_224",
    device=-1
)

def handler(params, context):
    image_base64 = params["image_base64"]

    result = classifier(image_base64)[0]

    return {
        "label": result["label"],
        "confidence": float(result["score"])
    }
