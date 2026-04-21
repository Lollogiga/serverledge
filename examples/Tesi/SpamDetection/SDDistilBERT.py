from transformers import pipeline

# Falconsai/spam_classification — DistilBERT fine-tuned for spam detection.
# ~67 M parameters, ~97 % accuracy on mixed SMS + email spam benchmarks.
# Returns "spam" / "ham" labels.
_classifier = pipeline(
    "text-classification",
    model="Falconsai/spam_classification",
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
