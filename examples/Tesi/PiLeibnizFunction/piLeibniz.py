# Baseline: n molto grande → errore trascurabile (≈ 2e-7), considerato esatto.
FIXED_N = 1_000_000


def handler(params, context):
    return pi_leibniz(FIXED_N)


def pi_leibniz(n):
    """
    Calcola pi usando la serie di Leibniz con n termini.
    Ritorna il risultato come stringa.
    """
    if n <= 0:
        return "0.0"

    s = 0.0
    sign = 1.0
    denom = 1.0

    for _ in range(n):
        s += sign / denom
        sign = -sign
        denom += 2.0

    return str(4.0 * s)
