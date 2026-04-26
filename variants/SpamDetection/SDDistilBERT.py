from transformers import pipeline

# mariagrandury/distilbert-base-uncased-finetuned-sms-spam-detection
# DistilBERT fine-tuned for SMS spam detection.
# ~67 M parameters. Returns LABEL_1 (spam) / LABEL_0 (ham).
_classifier = pipeline(
    "text-classification",
    model="mariagrandury/distilbert-base-uncased-finetuned-sms-spam-detection",
    device=-1,
)

_SPAM_LABELS = {"SPAM", "LABEL_1", "1"}


def handler(params, context):
    text = params["text"]
    result = _classifier(text)[0]
    raw = result["label"].upper()
    label = "SPAM" if raw in _SPAM_LABELS else "HAM"
    return {
        "label": label,
        "confidence": float(result["score"]),
        "model": "distilbert-spam",
    }
