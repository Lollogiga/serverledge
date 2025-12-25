def handler(params, context):
    n = int(params["n"])
    eps = 1.0 / (2 * n + 1)
    return pi_leibniz_approx(eps)


def pi_leibniz_approx(eps):
    if eps <= 0:
        return "0.0"

    s = 0.0
    sign = 1.0
    k = 0

    while True:
        denom = 2 * k + 1
        term = sign / denom
        s += term

        if abs(term) < eps:
            break

        sign = -sign
        k += 1

    return str(4.0 * s)
