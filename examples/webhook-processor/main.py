"""Webhook Processor — Async webhook handling with Runqy.

This example demonstrates a common pattern:
- Receive webhooks via your API
- Enqueue them for reliable async processing
- Handle retries automatically on failure

Use cases: Stripe payments, GitHub events, Slack commands, etc.
"""

import json
import sys
import urllib.request
from runqy_python import task, run


@task
def process(payload: dict) -> dict:
    """Process an incoming webhook event.

    Input:
        source (str): Webhook source (e.g., "stripe", "github")
        event_type (str): Event type (e.g., "payment.completed")
        data (dict): Webhook payload data

    Output:
        processed (bool): Whether the event was handled
        action (str): What action was taken
    """
    source = payload.get("source", "unknown")
    event_type = payload.get("event_type", "")
    data = payload.get("data", {})

    print(f"Processing {source} webhook: {event_type}", file=sys.stderr)

    # Route by source and event type
    if source == "stripe":
        return handle_stripe(event_type, data)
    elif source == "github":
        return handle_github(event_type, data)
    else:
        return {"processed": False, "action": f"Unknown source: {source}"}


def handle_stripe(event_type: str, data: dict) -> dict:
    """Handle Stripe webhook events."""
    if event_type == "payment_intent.succeeded":
        amount = data.get("amount", 0)
        customer = data.get("customer", "unknown")
        # Your business logic here: update DB, send receipt, etc.
        return {
            "processed": True,
            "action": f"Payment ${amount/100:.2f} from {customer} recorded",
        }
    elif event_type == "customer.subscription.deleted":
        return {
            "processed": True,
            "action": f"Subscription cancelled for {data.get('customer')}",
        }
    return {"processed": False, "action": f"Unhandled Stripe event: {event_type}"}


def handle_github(event_type: str, data: dict) -> dict:
    """Handle GitHub webhook events."""
    if event_type == "push":
        repo = data.get("repository", "unknown")
        branch = data.get("branch", "unknown")
        return {
            "processed": True,
            "action": f"Push to {repo}/{branch} — triggering build",
        }
    elif event_type == "issue.opened":
        return {
            "processed": True,
            "action": f"New issue #{data.get('number')}: {data.get('title')}",
        }
    return {"processed": False, "action": f"Unhandled GitHub event: {event_type}"}


if __name__ == "__main__":
    run()
