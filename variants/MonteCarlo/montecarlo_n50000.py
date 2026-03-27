import random

# Monte Carlo Pi estimation with N=50000 samples.
# Error (1σ) ≈ 4·√(p(1−p)/N) ≈ 0.0074
FIXED_N = 50_000


def handler(params, context):
    inside = 0
    for _ in range(FIXED_N):
        x = random.random()
        y = random.random()
        if x * x + y * y <= 1.0:
            inside += 1
    return str(4.0 * inside / FIXED_N)
