import os
from langchain_core.documents import Document
from langchain_ollama import OllamaEmbeddings
from langchain_community.llms import Ollama
from langchain_community.vectorstores import FAISS
from langchain.chains import RetrievalQA

CHUNKS_FOLDER = "chunks"

def load_chunks():
    print("Recursively loading .txt chunks...")
    docs = []
    for root, _, files in os.walk(CHUNKS_FOLDER):
        for filename in files:
            if filename.endswith(".txt"):
                path = os.path.join(root, filename)
                try:
                    with open(path, "r", encoding="utf-8") as f:
                        content = f.read().strip()
                except UnicodeDecodeError:
                    try:
                        with open(path, "r", encoding="latin-1") as f:
                            content = f.read().strip()
                        print(f"Non-UTF8 file read with latin-1: {os.path.relpath(path, CHUNKS_FOLDER)}")
                    except Exception as e:
                        print(f"Skipping unreadable file: {os.path.relpath(path, CHUNKS_FOLDER)} - {e}")
                        continue
                if content:
                    docs.append(Document(
                        page_content=content,
                        metadata={"source": os.path.relpath(path, CHUNKS_FOLDER)}
                    ))
                else:
                    print(f"Skipped empty file: {os.path.relpath(path, CHUNKS_FOLDER)}")
    print(f"Loaded {len(docs)} non-empty documents.")
    return docs

def main():
    docs = load_chunks()
    
    # Use OllamaEmbeddings from langchain_ollama (updated package)
    embeddings = OllamaEmbeddings(model="llama3.2:1b")
    
    # Build vector store with FAISS
    vectorstore = FAISS.from_documents(docs, embeddings)
    
    # Initialize Ollama LLM from langchain_community.llms
    llm = Ollama(model="llama3.2:1b")
    
    # Setup Retrieval QA chain
    qa = RetrievalQA.from_chain_type(llm=llm, retriever=vectorstore.as_retriever())
    
    query = "Your question here"
    answer = qa.run(query)
    print("Answer:", answer)

if __name__ == "__main__":
    main()
