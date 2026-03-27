from transformers import pipeline

# BERT-tiny fine-tuned on SST-2.
# ~4.4M parameters — very lightweight transformer.
# Benchmark accuracy on SST-2: ~84%  (error_estimate ≈ 0.16)
_classifier = pipeline(
    "sentiment-analysis",
    model="mrm8488/bert-tiny-finetuned-sst2",
    device=-1,
)

_LABEL_MAP = {"LABEL_0": "NEGATIVE", "LABEL_1": "POSITIVE"}


def handler(params, context):
    text = params["text"]
    result = _classifier(text)[0]
    label = _LABEL_MAP.get(result["label"], result["label"])
    return {
        "label": label,
        "confidence": float(result["score"]),
    }
