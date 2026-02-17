"""Data Pipeline — ETL tasks with Runqy.

This example demonstrates a simple ETL pattern:
- Fetch data from an external API
- Transform and enrich it
- Output structured results

Use cases: API data sync, report generation, data enrichment, scraping.
"""

import sys
import json
import urllib.request
from datetime import datetime
from runqy_python import task, run_once


@task
def process(payload: dict) -> dict:
    """Run a data pipeline step.

    Input:
        pipeline (str): Pipeline type ("fetch_users", "fetch_weather", "aggregate")
        params (dict): Pipeline-specific parameters

    Output:
        records (int): Number of records processed
        data (list): Processed data
        timestamp (str): Processing timestamp
    """
    pipeline = payload.get("pipeline", "")
    params = payload.get("params", {})

    print(f"Running pipeline: {pipeline}", file=sys.stderr)

    if pipeline == "fetch_users":
        return fetch_users(params)
    elif pipeline == "fetch_weather":
        return fetch_weather(params)
    elif pipeline == "aggregate":
        return aggregate(params)
    else:
        return {"error": f"Unknown pipeline: {pipeline}"}


def fetch_users(params: dict) -> dict:
    """Fetch and transform user data from JSONPlaceholder API."""
    limit = params.get("limit", 10)

    url = "https://jsonplaceholder.typicode.com/users"
    req = urllib.request.Request(url)
    with urllib.request.urlopen(req, timeout=10) as resp:
        users = json.loads(resp.read())

    # Transform: extract only what we need
    processed = [
        {
            "id": u["id"],
            "name": u["name"],
            "email": u["email"],
            "city": u["address"]["city"],
            "company": u["company"]["name"],
        }
        for u in users[:limit]
    ]

    return {
        "records": len(processed),
        "data": processed,
        "timestamp": datetime.utcnow().isoformat(),
    }


def fetch_weather(params: dict) -> dict:
    """Fetch weather data for a list of cities."""
    cities = params.get("cities", ["London"])

    results = []
    for city in cities:
        url = f"https://wttr.in/{city}?format=j1"
        try:
            req = urllib.request.Request(url)
            with urllib.request.urlopen(req, timeout=10) as resp:
                data = json.loads(resp.read())
                current = data["current_condition"][0]
                results.append({
                    "city": city,
                    "temp_c": current["temp_C"],
                    "description": current["weatherDesc"][0]["value"],
                })
        except Exception as e:
            results.append({"city": city, "error": str(e)})

    return {
        "records": len(results),
        "data": results,
        "timestamp": datetime.utcnow().isoformat(),
    }


def aggregate(params: dict) -> dict:
    """Aggregate data from multiple sources."""
    values = params.get("values", [])

    if not values:
        return {"error": "No values to aggregate"}

    return {
        "records": len(values),
        "data": {
            "count": len(values),
            "sum": sum(values),
            "avg": sum(values) / len(values),
            "min": min(values),
            "max": max(values),
        },
        "timestamp": datetime.utcnow().isoformat(),
    }


if __name__ == "__main__":
    run_once()
