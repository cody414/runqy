"""Scheduled Tasks — Cron-like periodic jobs with Runqy.

This example demonstrates scheduled/periodic task patterns:
- Health checks
- Report generation
- Cache warming
- Cleanup jobs

Enqueue these on a schedule using cron, systemd timers,
or the Runqy API from your application.
"""

import sys
import json
import urllib.request
from datetime import datetime
from runqy_python import task, run_once


@task
def process(payload: dict) -> dict:
    """Run a scheduled job.

    Input:
        job (str): Job type ("healthcheck", "report", "cleanup")
        config (dict): Job-specific configuration

    Output:
        status (str): Job result status
        details (dict): Job-specific output
        ran_at (str): Execution timestamp
    """
    job = payload.get("job", "")
    config = payload.get("config", {})

    print(f"[{datetime.utcnow().isoformat()}] Running scheduled job: {job}", file=sys.stderr)

    if job == "healthcheck":
        return healthcheck(config)
    elif job == "report":
        return daily_report(config)
    elif job == "cleanup":
        return cleanup(config)
    else:
        return {"status": "error", "details": {"error": f"Unknown job: {job}"}}


def healthcheck(config: dict) -> dict:
    """Check health of external services."""
    endpoints = config.get("endpoints", [
        {"name": "API", "url": "https://httpbin.org/status/200"},
        {"name": "DNS", "url": "https://dns.google/resolve?name=example.com"},
    ])

    results = []
    for ep in endpoints:
        try:
            req = urllib.request.Request(ep["url"], method="GET")
            with urllib.request.urlopen(req, timeout=5) as resp:
                results.append({
                    "name": ep["name"],
                    "status": "up",
                    "code": resp.status,
                })
        except Exception as e:
            results.append({
                "name": ep["name"],
                "status": "down",
                "error": str(e),
            })

    all_up = all(r["status"] == "up" for r in results)

    return {
        "status": "healthy" if all_up else "degraded",
        "details": {"checks": results},
        "ran_at": datetime.utcnow().isoformat(),
    }


def daily_report(config: dict) -> dict:
    """Generate a summary report."""
    # In practice: query your DB, aggregate metrics, etc.
    metrics = config.get("metrics", {
        "tasks_processed": 1547,
        "tasks_failed": 12,
        "avg_duration_ms": 234,
        "gpu_utilization": 0.73,
    })

    success_rate = 1 - (metrics["tasks_failed"] / max(metrics["tasks_processed"], 1))

    return {
        "status": "ok",
        "details": {
            "period": "daily",
            "metrics": metrics,
            "success_rate": f"{success_rate:.1%}",
            "summary": f"Processed {metrics['tasks_processed']} tasks, "
                       f"{success_rate:.1%} success rate, "
                       f"avg {metrics['avg_duration_ms']}ms",
        },
        "ran_at": datetime.utcnow().isoformat(),
    }


def cleanup(config: dict) -> dict:
    """Clean up old data, temp files, etc."""
    max_age_days = config.get("max_age_days", 30)

    # In practice: delete old records, clean temp storage, etc.
    return {
        "status": "ok",
        "details": {
            "max_age_days": max_age_days,
            "records_deleted": 42,
            "storage_freed_mb": 128,
        },
        "ran_at": datetime.utcnow().isoformat(),
    }


if __name__ == "__main__":
    run_once()
