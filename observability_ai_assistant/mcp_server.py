from fastmcp import FastMCP
from .tools.prometheus_tools import query_prometheus, get_active_alerts
from .tools.hawkeye_tools import get_health_checks, get_recent_incidents

# Initialize FastMCP server
mcp = FastMCP("ObservabilityAI")

@mcp.tool()
def query_metrics(promql: str) -> str:
    """Query Prometheus using PromQL."""
    try:
        data = query_prometheus(promql)
        return str(data)
    except Exception as e:
        return f"Error querying Prometheus: {str(e)}"

@mcp.tool()
def get_alerts() -> str:
    """Fetch active alerts from Alertmanager and Prometheus."""
    try:
        alerts = get_active_alerts()
        return str(alerts)
    except Exception as e:
        return f"Error fetching alerts: {str(e)}"

@mcp.tool()
def get_system_health() -> str:
    """Get status of health checks from Hawk-Eye engine."""
    try:
        health = get_health_checks()
        return str(health)
    except Exception as e:
        return f"Error fetching system health: {str(e)}"

@mcp.tool()
def get_incidents() -> str:
    """Get summarized incidents from Hawk-Eye engine."""
    try:
        incidents = get_recent_incidents()
        return str(incidents)
    except Exception as e:
        return f"Error fetching incidents: {str(e)}"

if __name__ == "__main__":
    mcp.run()
