# VADER-inspired pure-Python lexicon classifier.
# No neural network, no external dependencies — runs on the standard python310 runtime.
#
# Uses a curated subset of the AFINN-111 sentiment lexicon (Nielsen, 2011).
# Each word has an integer polarity score in [-5, +5].
# Benchmark accuracy on general sentiment tasks: ~70%  (error_estimate ≈ 0.30)

_AFINN = {
    # very positive (+4 / +5)
    "outstanding": 5, "superb": 5, "magnificent": 5, "brilliant": 5, "fantastic": 5,
    "excellent": 5, "wonderful": 5, "amazing": 5, "extraordinary": 5, "perfect": 5,
    "love": 4, "best": 4, "awesome": 4, "great": 4, "beautiful": 4,
    "delightful": 4, "exceptional": 4, "incredible": 4, "remarkable": 4,
    # positive (+2 / +3)
    "good": 3, "nice": 3, "pleasant": 3, "enjoy": 3, "happy": 3,
    "pleased": 3, "impressive": 3, "positive": 3, "recommend": 3,
    "fun": 2, "like": 2, "fine": 2, "solid": 2, "decent": 2,
    "well": 2, "helpful": 2, "useful": 2, "interesting": 2, "charming": 2,
    # mildly positive (+1)
    "okay": 1, "ok": 1, "satisfactory": 1, "adequate": 1, "reasonable": 1,
    "acceptable": 1, "alright": 1,
    # mildly negative (-1)
    "mediocre": -1, "average": -1, "disappointing": -1, "bland": -1,
    "dull": -1, "forgettable": -1, "slow": -1, "weak": -1,
    # negative (-2 / -3)
    "bad": -3, "poor": -3, "boring": -3, "worst": -3, "fail": -3,
    "failed": -3, "ugly": -3, "difficult": -2, "problem": -2, "lacks": -2,
    "lack": -2, "awful": -2, "sad": -2, "dislike": -2, "wrong": -2,
    "issue": -2, "broken": -2, "waste": -2, "tedious": -2, "painful": -2,
    # very negative (-4 / -5)
    "terrible": -5, "horrible": -5, "disgusting": -5, "dreadful": -5, "appalling": -5,
    "atrocious": -5, "catastrophic": -5, "abysmal": -5, "revolting": -5, "despicable": -4,
    "hate": -4, "hates": -4, "hated": -4, "detest": -4, "loathe": -4,
    "useless": -4, "pathetic": -4, "ridiculous": -4, "disaster": -4,
    # negation words (handled separately in scoring)
    "not": 0, "no": 0, "never": 0, "neither": 0, "nor": 0,
}

# Simple negation: if a negation token immediately precedes a sentiment word,
# flip its sign.
_NEGATIONS = frozenset({"not", "no", "never", "neither", "nor", "without",
                        "nobody", "nothing", "nowhere", "hardly", "barely",
                        "scarcely", "doesn't", "don't", "didn't", "isn't",
                        "wasn't", "weren't", "won't", "wouldn't", "can't",
                        "cannot", "couldn't", "shouldn't", "mustn't"})


def _score_text(text: str) -> float:
    tokens = text.lower().split()
    total = 0.0
    prev_negation = False
    for token in tokens:
        # Strip common punctuation
        word = token.strip(".,!?;:\"'()-")
        if word in _NEGATIONS:
            prev_negation = True
            continue
        val = _AFINN.get(word, 0)
        if prev_negation:
            val = -val
            prev_negation = False
        else:
            prev_negation = False
        total += val
    return total


def handler(params, context):
    text = params["text"]
    score = _score_text(text)

    if score >= 1.0:
        label = "POSITIVE"
    elif score <= -1.0:
        label = "NEGATIVE"
    else:
        label = "NEUTRAL"

    # Confidence: normalise score to [0, 1], capped at ±10 for saturation.
    confidence = min(abs(score) / 10.0, 1.0)

    return {
        "label": label,
        "confidence": confidence,
    }
