# Baseline: N molto grande → errore trascurabile, considerato esatto.
# Calcola ln(2) tramite la serie armonica alternante:
#   ln(2) = sum_{k=1}^{N} (-1)^(k+1) / k = 1 - 1/2 + 1/3 - 1/4 + ...
# Convergenza O(1/N): stessa classe della serie di Leibniz per π.
FIXED_N = 1_000_000


def handler(params, context):
    return ln_harmonic(FIXED_N)


def ln_harmonic(n):
    if n <= 0:
        return "0.0"
    s = 0.0
    sign = 1.0
    for k in range(1, n + 1):
        s += sign / k
        sign = -sign
    return str(s)
