from transformers import pipeline

# DistilBERT fine-tuned on SST-2 (Stanford Sentiment Treebank).
# ~67M parameters — roughly 5× lighter than RoBERTa-large.
# Benchmark accuracy on SST-2: ~91.3%  (error_estimate ≈ 0.087)
_classifier = pipeline(
    "sentiment-analysis",
    model="distilbert-base-uncased-finetuned-sst-2-english",
    device=-1,
)


def handler(params, context):
    text = params["text"]
    result = _classifier(text)[0]
    return {
        "label": result["label"],
        "confidence": float(result["score"]),
    }
