def handler(params, context):
    n_float = float(params["n"])

    if n_float < 0:
        return {"error": "err"} # Messaggio breve per risparmiare byte

    n = int(n_float)

    if n == 0:
        return {"value": 0}

    result = 1 << (n.bit_length() >> 1)

    return {
        "value": result,
    }