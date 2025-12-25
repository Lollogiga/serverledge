FROM python:3.10-slim

WORKDIR /

RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip

RUN pip install --no-cache-dir \
    "numpy<2" \
    torch==2.2.2+cpu \
    transformers==4.38.2 \
    sentencepiece \
    --extra-index-url https://download.pytorch.org/whl/cpu

COPY images/python-ml/executor.py /executor.py
RUN mkdir -p /app

CMD ["python3", "/executor.py"]
