# Dockerfile for AI Observability Assistant
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY . .

# Expose port for web API
EXPOSE 8000

# Default command to run MCP server
CMD ["python", "-m", "observability_ai_assistant.mcp_server"]