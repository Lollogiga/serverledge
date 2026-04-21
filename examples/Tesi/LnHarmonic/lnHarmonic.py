import math


def handler(params, context):
    """
    Calcola ln(2) esatto tramite math.log.
    Usato come baseline (variante non approssimata) per la selezione Pareto.
    """
    return str(math.log(2))
