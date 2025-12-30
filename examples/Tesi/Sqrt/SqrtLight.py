def handler(params, context):
    n = float(params["n"])

    if n < 0:
        return {"error": "sqrt not defined for negative numbers"}

    if n == 0:
        return {
            "value": 0.0,
        }

    # Numero di iterazioni scelto per garantire errore relativo ~1e-6
    ITERATIONS = 5

    x = n / 2.0

    for _ in range(ITERATIONS):
        x = 0.5 * (x + n / x)

    return {
        "value": x,
    }
