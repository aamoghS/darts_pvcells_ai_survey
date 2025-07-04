#!/bin/bash

echo "Filtering the PDF pages to specifics"
python3 filter_pdf/filterPage.py

echo "Extracting the sample dataset"
go run sort_and_rename.go

echo "Assigning the dataset"
python3 assign_dataset/assignData.py

echo "Run the model"
python3 model/script.py