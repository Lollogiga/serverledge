from transformers import pipeline

classifier = pipeline(
        "sentiment-analysis",
        model="siebert/sentiment-roberta-large-english",
        device=-1
    )

def handler(params, context):
    text = params["text"]

    result = classifier(text)[0]

    return {
        "label": result["label"],
        "confidence": float(result["score"])
    }
