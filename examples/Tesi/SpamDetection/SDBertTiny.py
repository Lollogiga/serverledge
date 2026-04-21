from transformers import pipeline

# mrm8488/bert-tiny-finetuned-sms-spam-detection
# ~4.4 M parameters (2-layer BERT-tiny), fine-tuned on UCI SMS Spam Collection.
# Accuracy ~95 % on SMS spam test set.
# Returns LABEL_0 (ham) / LABEL_1 (spam).
_classifier = pipeline(
    "text-classification",
    model="mrm8488/bert-tiny-finetuned-sms-spam-detection",
    device=-1,
)

_SPAM_LABELS = {"LABEL_1", "SPAM", "1"}


def handler(params, context):
    text = params["text"]
    result = _classifier(text)[0]
    raw = result["label"].upper()
    label = "SPAM" if raw in _SPAM_LABELS else "HAM"
    return {
        "label": label,
        "confidence": float(result["score"]),
        "model": "bert-tiny-spam",
    }
