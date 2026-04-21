from transformers import pipeline

# mshenoda/roberta-spam — RoBERTa-base fine-tuned on Enron + SMS spam corpora.
# ~125 M parameters. Accuracy ~99.5 % on held-out spam test sets.
# Returns LABEL_0 (ham) / LABEL_1 (spam).
_classifier = pipeline(
    "text-classification",
    model="mshenoda/roberta-spam",
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
        "model": "roberta-spam",
    }
