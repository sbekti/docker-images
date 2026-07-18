#!/usr/bin/env python3
"""Hermes MCP tools for Splitwise and local integrations."""

from __future__ import annotations

import os
from typing import Any

from mcp.server.fastmcp import FastMCP

from skills.splitwise.splitwise_client import (
    DEFAULT_CURRENCY,
    SplitwiseError,
    create_expense as splitwise_api_create_expense,
    get_current_user as splitwise_api_get_current_user,
    get_expense as splitwise_api_get_expense,
    list_categories as splitwise_api_list_categories,
    list_friends as splitwise_api_list_friends,
    list_groups as splitwise_api_list_groups,
)

HOST = os.environ.get("HERMES_TOOLS_MCP_HOST", "0.0.0.0")
PORT = int(os.environ.get("HERMES_TOOLS_MCP_PORT", "9101"))

mcp = FastMCP("hermes_tools", host=HOST, port=PORT)


def _splitwise_error(exc: Exception) -> dict[str, Any]:
    if isinstance(exc, SplitwiseError):
        return exc.to_dict()
    return {"error": str(exc)}


@mcp.tool()
def splitwise_get_current_user() -> dict[str, Any]:
    """Return the authenticated Splitwise user."""
    try:
        return splitwise_api_get_current_user()
    except Exception as exc:
        return _splitwise_error(exc)


@mcp.tool()
def splitwise_list_groups() -> dict[str, Any]:
    """Return Splitwise groups and member IDs."""
    try:
        return {"groups": splitwise_api_list_groups()}
    except Exception as exc:
        return _splitwise_error(exc)


@mcp.tool()
def splitwise_list_friends() -> dict[str, Any]:
    """Return Splitwise friends and balances."""
    try:
        return {"friends": splitwise_api_list_friends()}
    except Exception as exc:
        return _splitwise_error(exc)


@mcp.tool()
def splitwise_list_categories() -> dict[str, Any]:
    """Return Splitwise categories and subcategory IDs."""
    try:
        return {"categories": splitwise_api_list_categories()}
    except Exception as exc:
        return _splitwise_error(exc)


@mcp.tool()
def splitwise_get_expense(expense_id: int) -> dict[str, Any]:
    """Return a Splitwise expense by ID."""
    try:
        return splitwise_api_get_expense(expense_id)
    except Exception as exc:
        return _splitwise_error(exc)


@mcp.tool()
def splitwise_create_expense(
    cost: str,
    description: str,
    group_id: int,
    users: list[dict[str, Any]],
    confirmed: bool,
    currency: str = DEFAULT_CURRENCY,
    date: str | None = None,
    category_id: int | None = None,
    details: str | None = None,
) -> dict[str, Any]:
    """Create a confirmed expense with explicit payer and participant shares."""
    if not confirmed:
        return {"error": "splitwise_create_expense requires confirmed=true after explicit user confirmation"}
    try:
        return splitwise_api_create_expense(
            cost=cost,
            description=description,
            group_id=group_id,
            users=users,
            currency=currency,
            date=date,
            category_id=category_id,
            details=details,
        )
    except Exception as exc:
        return _splitwise_error(exc)


if __name__ == "__main__":
    mcp.run(transport="streamable-http")
