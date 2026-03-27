from transformers import pipeline

# sshleifer/distilbart-cnn-12-6 — distilled BART, 12 encoder / 6 decoder layers.
# ~306M parameters, ~75% of BART-large speed-up. ROUGE-L ≈ 38.6 on CNN/DM.
_summarizer = pipeline(
    "summarization",
    model="sshleifer/distilbart-cnn-12-6",
    device=-1,
)


def handler(params, context):
    text = params.get("text", "")
    if not text:
        return {"error": "text parameter required"}
    result = _summarizer(text, max_length=130, min_length=30, do_sample=False)
    return {
        "summary": result[0]["summary_text"],
        "model": "distilbart-cnn-12-6",
    }
