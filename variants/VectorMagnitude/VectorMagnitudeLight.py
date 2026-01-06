def handler(params, context):
    x = float(params["x"])
    y = float(params["y"])

    # Prendiamo i valori assoluti per lavorare nel primo quadrante
    abs_x = abs(x)
    abs_y = abs(y)

    # Troviamo il cateto massimo e minimo (solo confronti, niente calcoli complessi)
    maximum = max(abs_x, abs_y)
    minimum = min(abs_x, abs_y)

    # Coefficienti ottimali per minimizzare l'errore medio:
    # Alpha ~ 0.960
    # Beta  ~ 0.398
    alpha = 0.960
    beta = 0.398

    # Formula: Ipotenusa ~= (Alpha * Max) + (Beta * Min)
    # Nessuna radice quadrata, nessuna elevazione a potenza.
    result = (alpha * maximum) + (beta * minimum)

    return {
        "magnitude": result,
        "method": "alpha_max_plus_beta"
    }