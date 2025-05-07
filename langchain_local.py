# -*- coding: utf-8 -*-
"""
Created on Sun May  4 08:43:44 2025

@author: Doing
"""

from langchain_ollama import OllamaLLM

llm = OllamaLLM(model="llama3.2:3b")

answer = llm.invoke("What is a solar panels efficiency ...")

print(answer)