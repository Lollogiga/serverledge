import os
import json
import math

def handler(params, context):
    # Recupero input dall'utente
    x = float(params["x"])

    # Parametri che differiscono nelle varianti
    epsilon = float(os.getenv("APPROX_EPSILON", "1e-12"))
    max_iter = int(os.getenv("APPROX_MAX_ITER", "100"))
    variant = os.getenv("APPROX_VARIANT", "heavy")

    # Metodo di Newton
    guess = x / 2.0
    for i in range(max_iter):
        new_guess = 0.5 * (guess + x / guess)
        if abs(new_guess - guess) < epsilon:
            break
        guess = new_guess

    # Calcolo errore rispetto al valore reale
    true_value = math.sqrt(x)
    error = abs(true_value - guess)

    # Risposta JSON (Serverledge la serializza automaticamente)
    return {
        "input": x,
        "sqrt_estimate": guess,
        "true_value": true_value,
        "abs_error": error,
        "iterations": i + 1,
        "epsilon_used": epsilon,
        "max_iter_used": max_iter,
        "variant": variant
    }
