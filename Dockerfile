FROM python:3.10-slim

# System dependencies
RUN apt-get update && apt-get install -y \
    git \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy all project files into the container
COPY . .

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

# Expose the Marimo default port
EXPOSE 4000

# Run Marimo on your main file modify later 
CMD ["marimo", "run", "langchain_local.py", "--host", "0.0.0.0"] 
