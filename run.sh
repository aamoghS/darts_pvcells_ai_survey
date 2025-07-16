#!/bin/bash
set -e

source "$(conda info --base)/etc/profile.d/conda.sh"

if ! conda info --envs | awk '{print $1}' | grep -Fxq "pv-extraction-env"; then
    echo "creating"
    conda env create -f environment.yml
else
    echo "starting"
fi

conda activate pv-extraction-env
marimo edit PV-Work/mainScript/extraction.py
