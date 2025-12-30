import math

def handler(params, context):
    x = float(params["x"])
    y = float(params["y"])

    # Metodo standard: Pitagora
    # Sotto il cofano fa: sqrt(x*x + y*y)
    # math.hypot gestisce anche l'overflow meglio di farlo a mano,
    # ma è computazionalmente oneroso.
    result = math.hypot(x, y)

    return {
        "magnitude": result,
        "method": "euclidean_standard"
    }