def handler(params, context):
    x = float(params["x"])
    epsilon = 1e-12  # alta accuratezza

    result, iters = newton_sqrt(x, epsilon)
    return {
        "sqrt": result,
        "iterations": iters
    }


def newton_sqrt(x, epsilon):
    r_old = x
    iterations = 0

    while True:
        r_new = 0.5 * (r_old + x / r_old)
        iterations += 1

        if abs(r_new - r_old) < epsilon:
            return r_new, iterations

        r_old = r_new
