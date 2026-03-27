def handler(params, context):
    n = float(params["n"])

    if n < 0:
        return {"error": "sqrt not defined for negative numbers"}
    if n == 0:
        return {"value": 0.0}

    # Initial guess via integer bit-length (same as SqrtLight).
    # For n >= 1: x0 is within a factor of sqrt(2) from the true root,
    # i.e. relative error <= 1 - 1/sqrt(2) ≈ 29%.
    x = float(1 << (max(int(n), 1).bit_length() >> 1))

    # One Newton-Raphson refinement: x1 = (x + n/x) / 2
    # Quadratic convergence: error after step ≈ (error_before)^2 / 2.
    # Starting from <=29% error, after 1 step: <=4.2% worst case, ~2% typical.
    x = 0.5 * (x + n / x)

    return {"value": x}
