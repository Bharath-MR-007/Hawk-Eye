import requests
from typing import Any, Dict, List
from ..config import HAWKEYE_URL

def get_health_checks() -> List[Dict[str, Any]]:
    """Fetch health check results from Hawk-Eye."""
    # Assuming standard inventory/health endpoint
    response = requests.get(f"{HAWKEYE_URL}/inventory")
    response.raise_for_status()
    return response.json()

def get_recent_incidents() -> List[Dict[str, Any]]:
    """Fetch incidents from Hawk-Eye engine."""
    response = requests.get(f"{HAWKEYE_URL}/api/incidents")
    response.raise_for_status()
    return response.json()

def run_traceroute(target: str) -> Dict[str, Any]:
    """Trigger or retrieve traceroute for a target."""
    # This might be a POST or a GET depending on the setup
    response = requests.get(f"{HAWKEYE_URL}/api/traceroute", params={"target": target})
    response.raise_for_status()
    return response.json()
