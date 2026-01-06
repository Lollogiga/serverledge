def handler(params, context):
    n = int(params["n"])
    return fibonacci_nums_opt(n)


def fibonacci_nums_opt(n):
    if n <= 0:
        return "0"

    # Accumulo in lista (O(1) append)
    fib = [0, 1]

    for _ in range(2, n + 1):
        fib.append(fib[-1] + fib[-2])

    return ",".join(map(str, fib))
