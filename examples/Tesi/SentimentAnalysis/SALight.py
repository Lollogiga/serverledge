from transformers import pipeline

LABEL_MAP = {
    "LABEL_0": "NEGATIVE",
    "LABEL_1": "NEUTRAL",
    "LABEL_2": "POSITIVE"
}

classifier = pipeline(
    "sentiment-analysis",
    model="cardiffnlp/twitter-roberta-base-sentiment",
    device=-1
)

def handler(params, context):
    text = params["text"]
    result = classifier(text)[0]

    return {
        "label": LABEL_MAP.get(result["label"], result["label"]),
        "confidence": float(result["score"])
    }