def handler(params, context):
    n = params["n"]
    return ''.join(pi_leibniz(int(n)))


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
    count = 0

    while count < n:
        s += sign / denom
        sign = -sign
        denom += 2.0
        count += 1

    pi_val = 4.0 * s
    return str(pi_val)
