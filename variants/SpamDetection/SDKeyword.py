import re

# Keyword-based spam heuristic — no ML dependencies, runs on python310 runtime.
# Matches against a curated spam keyword/pattern list; scores each message.
# Accuracy ~72 % on mixed SMS + email corpora (high false-negative rate on
# context-dependent phishing messages that lack explicit spam words).

_SPAM_KEYWORDS = frozenset([
    "free", "win", "winner", "won", "prize", "claim",
    "limited time", "act now", "urgent", "congratulations",
    "cash", "lottery", "reward", "selected", "exclusive offer",
    "discount", "100%", "guarantee", "credit", "debt", "loan",
    "investment", "million", "billion", "bitcoin", "crypto",
    "unsubscribe", "opt out", "reply stop",
    "verify your account", "update your information",
    "confirm your", "click here", "click below",
    "earn money", "make money", "work from home",
])

_SPAM_PATTERNS = [
    re.compile(r'\$[\d,]+', re.IGNORECASE),          # monetary amounts
    re.compile(r'\bfree\b', re.IGNORECASE),
    re.compile(r'\bwin(ner)?\b', re.IGNORECASE),
    re.compile(r'click\s+here', re.IGNORECASE),
    re.compile(r'\burgent\b', re.IGNORECASE),
    re.compile(r'limited\s+time', re.IGNORECASE),
    re.compile(r'act\s+now', re.IGNORECASE),
    re.compile(r'congratul', re.IGNORECASE),
    re.compile(r'\bclaim\s+(now|your)', re.IGNORECASE),
    re.compile(r'\b(earn|make)\s+\$', re.IGNORECASE),
]


def _score(text: str) -> int:
    text_lower = text.lower()
    score = sum(1 for kw in _SPAM_KEYWORDS if kw in text_lower)
    score += sum(1 for pat in _SPAM_PATTERNS if pat.search(text))
    return score


def handler(params, context):
    text = params["text"]
    score = _score(text)
    is_spam = score >= 2
    if is_spam:
        confidence = min(0.55 + score * 0.08, 0.95)
    else:
        confidence = max(0.90 - score * 0.10, 0.55)
    return {
        "label": "SPAM" if is_spam else "HAM",
        "confidence": confidence,
        "model": "keyword-heuristic",
    }
