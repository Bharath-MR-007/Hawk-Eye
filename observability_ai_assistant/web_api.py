from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import ollama
import uvicorn
import logging
import os
from typing import Optional
import re
import requests
from config import HAWKEYE_URL

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("observability-ai")

app = FastAPI(title="Observability Assistant API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Configuration
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "llama3.2:latest")

class ChatRequest(BaseModel):
    message: str
    context: Optional[str] = ""

class ChatResponse(BaseModel):
    response: str

@app.get("/health")
async def health():
    return {"status": "ok", "model": OLLAMA_MODEL}

@app.post("/chat", response_model=ChatResponse)
async def chat_endpoint(request: ChatRequest):
    logger.info(f"Received message: {request.message}")
    
    try:
        target_model = OLLAMA_MODEL
        
        # System Prompt - Anti-hallucination hardening
        system_prompt = """You are the Hawk-Eye Observability AI. 
        You analyze system incidents and targets using provided JSON data.
        
        CRITICAL RULES:
        1. If the user asks about a specific ID (inc- or trg-) and the SYSTEM CONTEXT is EMPTY, you MUST state: "I couldn't find data for that ID in my local cache. Please make sure the browser page is refreshed and the incident is visible on your screen."
        2. DO NOT make up incident details (like DNS errors or placeholder IPs) if they are not in the context.
        3. If context IS provided, use it to explain the specific alert status, severity, and description.
        4. Provide actionable troubleshooting steps relevant to the actual alert (e.g., if latency is high, suggest network checks).
        """

        full_message = request.message
        msg_lower = request.message.lower()

        # 1. Direct Backend Tool Execution: List Targets
        if "what targets" in msg_lower or "list targets" in msg_lower or "show targets" in msg_lower:
            try:
                res = requests.get(f"{HAWKEYE_URL}/api/v1/targets", cookies={"hawk_session": "authenticated"}, timeout=3)
                if res.status_code == 200:
                    targets = res.json()
                    if not targets:
                        return ChatResponse(response="Currently, there are **no targets** being monitored by Hawk-Eye.")
                    tls = "\n".join([f"- `{t}`" for t in targets])
                    return ChatResponse(response=f"Here is the list of active targets currently monitored by Hawk-Eye:\n{tls}\n\nAsk me to 'add target [URL]' to monitor a new one.")
            except Exception as e:
                logger.error(f"Failed list targets: {e}")

        # 2. Direct Backend Tool Execution: Add New Target
        if "add target" in msg_lower or "add new target" in msg_lower:
            # Extract standard URL or domain
            url_match = re.search(r'(https?://[^\s>]+|[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(?:/[^\s>]*)?)', request.message, re.IGNORECASE)
            if url_match:
                target_url = url_match.group(1).rstrip('.')
                try:
                    res = requests.post(f"{HAWKEYE_URL}/api/v1/targets", json={"url": target_url}, cookies={"hawk_session": "authenticated"}, allow_redirects=False, timeout=5)
                    if res.status_code in [200, 201]:
                        return ChatResponse(response=f"✅ Automatically done! I have successfully added `{target_url}` to the Hawk-Eye monitoring engine. You will see it appear in your dashboard shortly.")
                    else:
                        return ChatResponse(response=f"❌ I attempted to add `{target_url}`, but the Hawk-Eye engine returned an error: {res.status_code}.")
                except Exception as ex:
                    logger.error(f"Failed to add target: {ex}")
                    return ChatResponse(response=f"⚠️ I attempted to add `{target_url}` but couldn't reach the backend API at `{HAWKEYE_URL}`.")

        # 3. Handle standard context injection
        if request.context and "CRITICAL" in request.context:
            logger.info(f"Injecting verified context for: {request.message}")
            full_message = f"SYSTEM CONTEXT (Observability Data):\n{request.context}\n\nUSER QUESTION: {request.message}"
        elif any(x in request.message.lower() for x in ["inc-", "trg-", "incident", "target"]):
             # User asked for an ID but context search failed in JS
             full_message = f"USER QUESTION: {request.message}\n\nNOTE: System context for this ID was NOT found in local storage."

        response = ollama.chat(model=target_model, messages=[
            {'role': 'system', 'content': system_prompt},
            {'role': 'user', 'content': full_message},
        ])
        
        reply = response['message']['content']
        return ChatResponse(response=reply)
        
    except Exception as e:
        logger.error(f"Error in chat endpoint: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
