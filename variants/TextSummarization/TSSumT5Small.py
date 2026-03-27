from transformers import pipeline

# google/t5-small — 60M parameter T5 variant, abstractive summarisation.
# Trained on C4; ROUGE-L ≈ 30.1 on CNN/DM.
_summarizer = pipeline(
    "summarization",
    model="google/t5-small",
    device=-1,
)


def handler(params, context):
    text = params.get("text", "")
    if not text:
        return {"error": "text parameter required"}
    result = _summarizer(
        "summarize: " + text,
        max_length=100,
        min_length=15,
        do_sample=False,
    )
    return {
        "summary": result[0]["summary_text"],
        "model": "t5-small",
    }
