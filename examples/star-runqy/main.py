#!/usr/bin/env python3
"""
Star Runqy Example - Vault Tutorial

This example shows how to use Runqy's built-in vault to securely manage GitHub tokens
and star the Publikey/runqy repository from within a Runqy task.

The GitHub token is provided via environment variable from the vault.
"""

import os
import requests
from runqy import task


@task
def star_runqy() -> dict:
    """
    Star the Publikey/runqy repository using the GitHub API.
    
    This task demonstrates:
    1. Using environment variables from the vault
    2. Making authenticated GitHub API calls
    3. Handling API responses
    4. Returning structured results
    
    Returns:
        dict: Result of the starring operation
    """
    # Get GitHub token from environment (injected by vault)
    github_token = os.getenv('GITHUB_TOKEN')
    if not github_token:
        return {
            "error": "GITHUB_TOKEN not found in environment",
            "success": False,
            "message": "Make sure the GitHub token is configured in the vault"
        }
    
    # GitHub API endpoint to star a repository
    repo_owner = "Publikey"
    repo_name = "runqy"
    star_url = f"https://api.github.com/user/starred/{repo_owner}/{repo_name}"
    
    # Headers for GitHub API authentication
    headers = {
        "Authorization": f"token {github_token}",
        "Accept": "application/vnd.github.v3+json",
        "User-Agent": "Runqy-Star-Example/1.0"
    }
    
    try:
        # First, check if already starred
        check_response = requests.get(star_url, headers=headers)
        
        if check_response.status_code == 204:
            return {
                "success": True,
                "already_starred": True,
                "message": "🌟 Repository was already starred! You're awesome!",
                "emoji_message": "You just starred Runqy from a Runqy task! 🌀"
            }
        
        # Star the repository
        star_response = requests.put(star_url, headers=headers)
        
        if star_response.status_code == 204:
            return {
                "success": True,
                "already_starred": False,
                "message": "🌟 Successfully starred Publikey/runqy! Thank you for the support!",
                "emoji_message": "You just starred Runqy from a Runqy task! 🌀",
                "repo_url": f"https://github.com/{repo_owner}/{repo_name}"
            }
        else:
            return {
                "error": f"Failed to star repository: {star_response.status_code}",
                "success": False,
                "response_text": star_response.text[:200]  # First 200 chars for debugging
            }
    
    except requests.RequestException as e:
        return {
            "error": f"Network error: {str(e)}",
            "success": False
        }
    except Exception as e:
        return {
            "error": f"Unexpected error: {str(e)}",
            "success": False
        }


if __name__ == "__main__":
    # For local testing (when token is in environment)
    result = star_runqy()
    print("🌀 Star Runqy Result:")
    print(f"Success: {result.get('success')}")
    print(f"Message: {result.get('message', result.get('emoji_message', 'No message'))}")
    
    if 'error' in result:
        print(f"Error: {result['error']}")