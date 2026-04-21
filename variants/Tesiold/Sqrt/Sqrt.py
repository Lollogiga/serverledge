import math

def handler(params, context):
    n = float(params["n"])

    if n < 0:
        return {"error": "sqrt not defined for negative numbers"}

    result = math.sqrt(n)

    return {
        "value": result,
    }
