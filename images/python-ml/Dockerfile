FROM python:3.10-slim

WORKDIR /

# =====================================================
# DIPENDENZE DI SISTEMA (per PIL)
# =====================================================
RUN apt-get update && apt-get install -y \
    git \
    libjpeg-dev \
    zlib1g-dev \
    libpng-dev \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip

# =====================================================
# DIPENDENZE PYTHON ML
# =====================================================
RUN pip install --no-cache-dir \
    "numpy<2" \
    torch==2.2.2+cpu \
    torchvision==0.17.2+cpu \
    transformers==4.38.2 \
    sentencepiece \
    pillow \
    --extra-index-url https://download.pytorch.org/whl/cpu

# =====================================================
# EXECUTOR SERVERLEDGE
# =====================================================
COPY images/python-ml/executor.py /executor.py
RUN mkdir -p /app

CMD ["python3", "/executor.py"]
