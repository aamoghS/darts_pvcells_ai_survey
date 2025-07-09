FROM python:3.10-slim

# System dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Install pip packages
RUN pip install --no-cache-dir \
    marimo==0.14.9 \
    pandas \
    pydantic \
    tqdm \
    pdfminer.six \
    langchain \
    langchain-community \
    datasets \
    peft \
    transformers \
    bitsandbytes-cpu \
    sentencepiece \
    accelerate \
    jupyter \
    matplotlib

CMD ["bash"]
