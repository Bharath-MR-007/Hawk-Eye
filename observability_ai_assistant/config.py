import os
from dotenv import load_dotenv

load_dotenv()

# API Endpoints
HAWKEYE_URL = os.getenv("HAWKEYE_URL", "http://localhost:8080")
PROMETHEUS_URL = os.getenv("PROMETHEUS_URL", "http://localhost:9090")
ALERTMANAGER_URL = os.getenv("ALERTMANAGER_URL", "http://localhost:9093")
GRAFANA_URL = os.getenv("GRAFANA_URL", "http://localhost:3000")
THANOS_QUERY_URL = os.getenv("THANOS_QUERY_URL", "http://localhost:10902")
ELASTICSEARCH_URL = os.getenv("ELASTICSEARCH_URL", "http://localhost:9200")

# LLM Configuration
LLM_PROVIDER = os.getenv("LLM_PROVIDER", "ollama")
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "llama3.2:latest")
OLLAMA_HOST = os.getenv("OLLAMA_HOST", "http://localhost:11434")

NVIDIA_MODEL = os.getenv("NVIDIA_MODEL", "deepseek-ai/deepseek-v3.2")
AVAILABLE_MODELS = os.getenv("AVAILABLE_MODELS", "deepseek-coder:latest,gemma4:latest,qwen2.5:14b").split(",")
