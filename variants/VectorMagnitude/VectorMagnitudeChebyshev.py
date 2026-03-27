def handler(params, context):
    x = float(params["x"])
    y = float(params["y"])

    # Chebyshev (L-infinity) approximation of the Euclidean (L2) norm.
    # Returns the largest absolute component as a lower bound for the magnitude.
    #
    # Error analysis for unit vector (cos θ, sin θ):
    #   ratio = max(|cos θ|, |sin θ|) / 1
    #   Minimum at θ = 45°: ratio = 1/√2 ≈ 0.707 → underestimates by 29.3%
    #   Maximum at θ = 0° / 90°: ratio = 1 → exact
    #
    # This is the cheapest possible approximation — one comparison, no multiply.
    result = max(abs(x), abs(y))

    return {
        "magnitude": result,
        "method": "chebyshev"
    }
