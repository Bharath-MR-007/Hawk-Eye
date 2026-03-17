import ollama
from typing import List, Dict, Any
from ..config import OLLAMA_MODEL

class OllamaClient:
    def __init__(self, model: str = OLLAMA_MODEL):
        self.model = model

    def chat(self, messages: List[Dict[str, str]]) -> str:
        """
        Send a chat request to the Ollama model.
        """
        response = ollama.chat(model=self.model, messages=messages)
        return response['message']['content']

    def generate(self, prompt: str) -> str:
        """
        Generate a completion for a given prompt.
        """
        response = ollama.generate(model=self.model, prompt=prompt)
        return response['response']
