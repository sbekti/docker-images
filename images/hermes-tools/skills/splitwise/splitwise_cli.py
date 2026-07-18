#!/usr/bin/env python3
"""CLI wrapper for the Hermes Splitwise skill client."""

from __future__ import annotations

import argparse
import json
import sys

from splitwise_client import (
    DEFAULT_CURRENCY,
    SplitwiseError,
    create_expense,
    get_current_user,
    get_expense,
    list_categories,
    list_friends,
    list_groups,
)


def _print(value: object) -> None:
    print(json.dumps(value, indent=2, sort_keys=True))


def _create(args: argparse.Namespace) -> dict[str, object]:
    return create_expense(
        cost=args.cost,
        description=args.description,
        group_id=args.group_id,
        users=json.loads(args.users),
        currency=args.currency,
        date=args.date,
        category_id=args.category_id,
        details=args.details,
    )


def main() -> int:
    parser = argparse.ArgumentParser(description="Splitwise CLI for Hermes")
    subcommands = parser.add_subparsers(dest="command", required=True)

    subcommands.add_parser("me", help="Get current user info")
    subcommands.add_parser("groups", help="List groups")
    subcommands.add_parser("friends", help="List friends with balances")
    subcommands.add_parser("categories", help="List expense categories")

    create = subcommands.add_parser("create-expense", help="Create an expense")
    create.add_argument("--cost", required=True, help="Total cost as a decimal string")
    create.add_argument("--description", required=True, help="Expense description")
    create.add_argument("--group-id", type=int, required=True, help="Group ID")
    create.add_argument("--users", required=True, help="JSON array of participant shares")
    create.add_argument("--currency", default=DEFAULT_CURRENCY, help="Currency code")
    create.add_argument("--date", help="ISO 8601 date")
    create.add_argument("--category-id", type=int, help="Splitwise category ID")
    create.add_argument("--details", help="Notes/details")

    get = subcommands.add_parser("get-expense", help="Get expense details")
    get.add_argument("expense_id", type=int)

    args = parser.parse_args()
    try:
        if args.command == "me":
            _print(get_current_user())
        elif args.command == "groups":
            _print(list_groups())
        elif args.command == "friends":
            _print(list_friends())
        elif args.command == "categories":
            _print(list_categories())
        elif args.command == "get-expense":
            _print(get_expense(args.expense_id))
        elif args.command == "create-expense":
            _print(_create(args))
        else:
            parser.error(f"unknown command {args.command}")
    except (SplitwiseError, json.JSONDecodeError) as exc:
        payload = exc.to_dict() if isinstance(exc, SplitwiseError) else {"error": str(exc)}
        print(json.dumps(payload, sort_keys=True), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
