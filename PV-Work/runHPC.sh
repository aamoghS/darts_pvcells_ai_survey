#!/bin/bash

echo "Sorting the files"
go run sort.go

echo "Extracting the sample dataset"
go run parser.go

echo "Assigning the datawset"
python3 assign_dataset/assignData.py

echo "Run the model"
python3 model/script.py
