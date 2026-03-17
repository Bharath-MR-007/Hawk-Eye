import sys
import json
from rich.console import Console
from rich.panel import Panel
from rich.markdown import Markdown
from .llm.ollama_client import OllamaClient
from .mcp_server import mcp

console = Console()

class ObservabilityAssistant:
    def __init__(self):
        self.llm = OllamaClient()
        self.system_prompt = """
        You are a highly skilled SRE and Observability Assistant. 
        Your goal is to help users diagnose system issues using metrics, logs, and alerts.
        
        You have access to the following tools via an MCP server:
        - query_metrics(promql): Query Prometheus for real-time data.
        - get_alerts(): Get current active alerts.
        - get_system_health(): Check Hawk-Eye health status.
        - get_incidents(): Get detailed incident logs.
        
        Follow this reasoning pipeline:
        1. Understand the user's question.
        2. Identify if you need data (metrics, alerts, health).
        3. Call the appropriate tools (you will simulate tool calls by describing them if necessary, 
           but in this direct implementation, I will provide the results).
        4. Correlate the findings and explain the root cause in plain language.
        """

    def handle_query(self, query: str):
        console.print(f"\n[bold purple]User:[/bold purple] {query}")
        
        # Initial thought process
        messages = [
            {"role": "system", "content": self.system_prompt},
            {"role": "user", "content": query}
        ]
        
        with console.status("[bold green]Thinking...") as status:
            first_response = self.llm.chat(messages)
            
            # Simple simulation of Tool Calling for this demonstration
            # In a full MCP client setup, we would parse ToolCall objects.
            # Here we provide the context the model asks for.
            
            if "query_metrics" in first_response or "get_alerts" in first_response or "health" in first_response:
                status.update("[bold cyan]Gathering observability data...")
                
                # We'll pull a snapshot of everything for a rich context update
                alerts = mcp.call_tool("get_alerts", {})
                health = mcp.call_tool("get_system_health", {})
                
                context_update = f"""
                OBSERVABILITY CONTEXT:
                Active Alerts: {alerts}
                System Health: {health}
                """
                
                messages.append({"role": "assistant", "content": first_response})
                messages.append({"role": "system", "content": context_update})
                
                status.update("[bold green]Analyzing and Correlating...")
                final_response = self.llm.chat(messages)
            else:
                final_response = first_response

        console.print(Panel(Markdown(final_response), title="[bold green]Assistant Result", border_style="green"))

def main():
    assistant = ObservabilityAssistant()
    
    console.print(Panel.fit(
        "[bold cyan]Observability AI Assistant[/bold cyan]\n"
        "Ask me about your system health, metrics, or alerts.",
        border_style="cyan"
    ))
    
    while True:
        try:
            user_input = console.input("\n[bold magenta]>[/bold magenta] ")
            if user_input.lower() in ["exit", "quit", "bye"]:
                break
            assistant.handle_query(user_input)
        except KeyboardInterrupt:
            break

if __name__ == "__main__":
    main()
