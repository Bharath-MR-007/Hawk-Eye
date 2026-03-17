import requests
from typing import Any, Dict, List
from ..config import PROMETHEUS_URL, ALERTMANAGER_URL

def query_prometheus(promql: str) -> Dict[str, Any]:
    """Execute a PromQL query."""
    response = requests.get(f"{PROMETHEUS_URL}/api/v1/query", params={"query": promql})
    response.raise_for_status()
    return response.json()

def query_prometheus_range(promql: str, start: str, end: str, step: str = "15s") -> Dict[str, Any]:
    """Execute a PromQL range query."""
    response = requests.get(
        f"{PROMETHEUS_URL}/api/v1/query_range",
        params={"query": promql, "start": start, "end": end, "step": step}
    )
    response.raise_for_status()
    return response.json()

def get_active_alerts() -> List[Dict[str, Any]]:
    """Fetch active alerts from Alertmanager."""
    response = requests.get(f"{ALERTMANAGER_URL}/api/v2/alerts")
    response.raise_for_status()
    return response.json()

def get_prometheus_alerts() -> List[Dict[str, Any]]:
    """Fetch active alerts directly from Prometheus."""
    response = requests.get(f"{PROMETHEUS_URL}/api/v1/alerts")
    response.raise_for_status()
    return response.json().get("data", {}).get("alerts", [])
