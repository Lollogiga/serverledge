from transformers import pipeline

def handler(params, context):
    text = params["text"]

    classifier = pipeline(
        "sentiment-analysis",
        model="bert-large-uncased",
        device=-1
    )

    result = classifier(text)[0]

    return {
        "label": result["label"],
        "confidence": float(result["score"])
    }
