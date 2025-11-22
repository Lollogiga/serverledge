def handler(params, context):
    print(f"Invoked inc with input: {params}")
    # Usa 'n' invece di 'input'
    return int(params["n"]) + 1