import httpx
from typing import List, Dict, Any
import logging
from config import NVIDIA_API_KEY, NVIDIA_BASE_URL, NVIDIA_MODEL

logger = logging.getLogger(__name__)

class NVIDIANIMClient:
    """Client for NVIDIA NIM API with DeepSeek-V3.2"""
    
    def __init__(self, api_key: str = NVIDIA_API_KEY, base_url: str = NVIDIA_BASE_URL, model: str = NVIDIA_MODEL):
        self.api_key = api_key
        self.base_url = base_url
        self.model = model
        self.client = httpx.Client(timeout=60.0)
    
    def chat(self, messages: List[Dict[str, str]]) -> str:
        """Send chat completion request to NVIDIA NIM"""
        
        endpoint = f"{self.base_url}/chat/completions"
        headers = {
            "Authorization": f"Bearer {self.api_key}",
            "Content-Type": "application/json"
        }
        
        payload = {
            "model": self.model,
            "messages": messages,
            "temperature": 0.2,
            "max_tokens": 1024,
            "stream": False
        }
        
        try:
            logger.info(f"NVIDIA NIM Request: URL={endpoint}, Model={self.model}")
            response = self.client.post(endpoint, json=payload, headers=headers)
            response.raise_for_status()
            data = response.json()
            return data["choices"][0]["message"]["content"]
        except httpx.HTTPStatusError as e:
            logger.error(f"NVIDIA API error: {e.response.text}")
            return f"Error from NVIDIA NIM: {e.response.text}"
        except Exception as e:
            logger.error(f"Unexpected error: {str(e)}")
            return f"Unexpected error connecting to NVIDIA NIM: {str(e)}"

    def close(self):
        self.client.close()
