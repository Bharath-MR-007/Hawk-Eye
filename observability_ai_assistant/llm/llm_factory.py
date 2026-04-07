from .ollama_client import OllamaClient
from .nvidia_client import NVIDIANIMClient
from config import LLM_PROVIDER

class LLMFactory:
    """Factory to create LLM clients based on configuration"""
    
    @staticmethod
    def get_client(provider: str = None):
        if provider is None:
            provider = LLM_PROVIDER
        
        if provider.lower() == "ollama":
            return OllamaClient()
        elif provider.lower() == "nvidia":
            return NVIDIANIMClient()
        else:
            # Default fallback
            return OllamaClient()
