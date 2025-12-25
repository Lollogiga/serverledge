from transformers import pipeline

classifier = pipeline(
    "sentiment-analysis",
    model="distilbert-base-uncased-finetuned-sst-2-english",
    device=-1
)

def handler(params, context):
    text = params["text"]
    result = classifier(text)[0]
    return {
        "label": result["label"],
        "confidence": float(result["score"])
    }
