from transformers import pipeline

# facebook/bart-large-cnn — 400M parameters, state-of-the-art abstractive summariser.
# Fine-tuned on CNN/DailyMail. ROUGE-L ≈ 40.9 on CNN/DM.
_summarizer = pipeline(
    "summarization",
    model="facebook/bart-large-cnn",
    device=-1,
)


def handler(params, context):
    text = params.get("text", "")
    if not text:
        return {"error": "text parameter required"}
    result = _summarizer(text, max_length=130, min_length=30, do_sample=False)
    return {
        "summary": result[0]["summary_text"],
        "model": "bart-large-cnn",
    }
