import os
import json

def handler(event):
    model = os.getenv("APPROX_MODEL", "resnet50")
    aug  = os.getenv("APPROX_AUG", "none")

    # MOCK inference
    result = {
        "model_used": model,
        "augmentations_used": aug,
        "prediction": "cat",    # mock
        "confidence": 0.98       # mock
    }

    return json.dumps(result)
