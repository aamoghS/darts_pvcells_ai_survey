#!/bin/bash
set -e

# Initialize conda
source "$(conda info --base)/etc/profile.d/conda.sh"

# Check if env exists, create if not
if ! conda info --envs | awk '{print $1}' | grep -Fxq "pv-extraction-env"; then
    echo "Environment not found. Creating it now..."
    conda env create -f environment.yml
else
    echo "Environment found. Activating..."
fi

# Activate the environment
conda activate pv-extraction-env

# Open the Marimo notebook in editor mode
marimo edit PV-Work/mainScript/extraction.py
