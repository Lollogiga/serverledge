from transformers import pipeline

def handler(params, context):
    text = params["text"]

    classifier = pipeline(
        "sentiment-analysis",
        model="distilbert-base-uncased-finetuned-sst-2-english",
        device=-1
    )

    result = classifier(text)[0]

    return {
        "label": result["label"],
        "confidence": float(result["score"])
    }
