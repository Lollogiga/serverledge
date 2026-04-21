FIXED_N = 10000


def handler(params, context):
    return ln_harmonic(FIXED_N)


def ln_harmonic(n):
    s = 0.0
    sign = 1.0
    for k in range(1, n + 1):
        s += sign / k
        sign = -sign
    return str(s)
