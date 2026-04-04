# AI Observability Assistant (Mixtral + MCP)

A local AI assistant for SREs to diagnose system health using Mixtral (Ollama) and the Model Context Protocol (MCP).

## Prerequisites

1. **Ollama**: Install and pull the Mixtral model.
   ```bash
   ollama pull mixtral
   ```
2. **Observability Stack**: Ensure Prometheus (9090) and Hawk-Eye (8080) are running.

## Installation

```bash
cd observability_ai_assistant
pip install -r requirements.txt
```

### Docker

Build and run the AI assistant in a container:

```bash
# Build the Docker image
docker build -t hawk-eye-ai-assistant .

# Run the MCP server
docker run -p 8000:8000 hawk-eye-ai-assistant

# Or run the CLI (if needed)
docker run -it hawk-eye-ai-assistant python -m observability_ai_assistant.assistant
```

## Running the Assistant

1. **Start the MCP Server**:
   ```bash
   # In one terminal
   python -m observability_ai_assistant.mcp_server
   ```

2. **Start the Assistant CLI**:
   ```bash
   # In another terminal
   python -m observability_ai_assistant.assistant
   ```

## Features

- **Multi-Source Analysis**: Correlates metrics from Prometheus with incident logs from Hawk-Eye.
- **Natural Language Diagnostics**: Ask "Why is network latency high?" or "Show current alerts".
- **Reasoning Pipeline**: 
  1. Detect anomaly 
  2. Inspect alerts 
  3. Analyze logs 
  4. Correlate metrics 
  5. Explain root cause.
