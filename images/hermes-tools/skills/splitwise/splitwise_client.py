"""Small Splitwise API client for the Hermes Splitwise MCP tools."""

from __future__ import annotations

from decimal import Decimal, InvalidOperation
import json
import os
import urllib.error
import urllib.parse
import urllib.request

API_BASE = "https://secure.splitwise.com/api/v3.0"
USER_AGENT = "hermes-splitwise-skill/1.0"
DEFAULT_CURRENCY = "USD"


class SplitwiseError(RuntimeError):
    """Raised for Splitwise API and validation failures."""

    def __init__(self, message: str, *, status: int | None = None, errors: object | None = None):
        super().__init__(message)
        self.status = status
        self.errors = errors

    def to_dict(self) -> dict[str, object]:
        payload: dict[str, object] = {"error": str(self)}
        if self.status is not None:
            payload["status"] = self.status
        if self.errors is not None:
            payload["errors"] = self.errors
        return payload


def _api_key() -> str:
    token = os.environ.get("SPLITWISE_API_KEY", "").strip()
    if not token:
        raise SplitwiseError("Splitwise authentication is not configured")
    return token


def _json_loads(raw: bytes) -> object:
    text = raw.decode("utf-8", errors="replace")
    if not text:
        return {}
    try:
        return json.loads(text)
    except json.JSONDecodeError as exc:
        raise SplitwiseError(f"Splitwise returned non-JSON response: {exc}") from exc


def _request(
    method: str,
    path: str,
    *,
    json_data: object | None = None,
    form_data: dict[str, str] | None = None,
) -> dict[str, object]:
    url = f"{API_BASE}/{path.lstrip('/')}"
    body: bytes | None = None
    headers = {
        "Accept": "application/json",
        "Authorization": f"Bearer {_api_key()}",
        "User-Agent": USER_AGENT,
    }

    if json_data is not None and form_data is not None:
        raise SplitwiseError("json_data and form_data are mutually exclusive")
    if json_data is not None:
        body = json.dumps(json_data).encode("utf-8")
        headers["Content-Type"] = "application/json"
    if form_data is not None:
        body = urllib.parse.urlencode(form_data).encode("utf-8")
        headers["Content-Type"] = "application/x-www-form-urlencoded"

    request = urllib.request.Request(url, data=body, headers=headers, method=method.upper())
    try:
        with urllib.request.urlopen(request, timeout=35) as response:
            payload = _json_loads(response.read())
    except urllib.error.HTTPError as exc:
        payload = _json_loads(exc.read())
        errors = payload.get("errors") if isinstance(payload, dict) else payload
        raise SplitwiseError("Splitwise request failed", status=exc.code, errors=errors) from exc
    except urllib.error.URLError as exc:
        raise SplitwiseError(f"Splitwise network error: {exc.reason}") from exc

    if not isinstance(payload, dict):
        raise SplitwiseError("Splitwise returned an unexpected response shape")
    return payload


def _money(value: object, field: str) -> Decimal:
    try:
        return Decimal(str(value))
    except (InvalidOperation, ValueError) as exc:
        raise SplitwiseError(f"{field} must be a decimal string") from exc


def _name(user: dict[str, object]) -> str:
    first = str(user.get("first_name") or "").strip()
    last = str(user.get("last_name") or "").strip()
    return f"{first} {last}".strip()


def _compact_user(user: dict[str, object]) -> dict[str, object]:
    return {
        "id": user.get("id"),
        "name": _name(user),
        "email": user.get("email"),
    }


def get_current_user() -> dict[str, object]:
    user = _request("GET", "get_current_user").get("user", {})
    return _compact_user(user if isinstance(user, dict) else {})


def list_groups() -> list[dict[str, object]]:
    payload = _request("GET", "get_groups")
    groups: list[dict[str, object]] = []
    for group in payload.get("groups", []):
        if not isinstance(group, dict) or group.get("id") == 0:
            continue
        members = [
            {"id": member.get("id"), "name": _name(member)}
            for member in group.get("members", [])
            if isinstance(member, dict)
        ]
        groups.append({"id": group.get("id"), "name": group.get("name"), "members": members})
    return groups


def list_friends() -> list[dict[str, object]]:
    payload = _request("GET", "get_friends")
    friends: list[dict[str, object]] = []
    for friend in payload.get("friends", []):
        if not isinstance(friend, dict):
            continue
        balances = [
            {"currency": item.get("currency_code"), "amount": item.get("amount")}
            for item in friend.get("balance", [])
            if isinstance(item, dict)
        ]
        friends.append({**_compact_user(friend), "balances": balances})
    return friends


def list_categories() -> list[dict[str, object]]:
    payload = _request("GET", "get_categories")
    categories: list[dict[str, object]] = []
    for category in payload.get("categories", []):
        if not isinstance(category, dict):
            continue
        subcategories = [
            {"id": subcategory.get("id"), "name": subcategory.get("name")}
            for subcategory in category.get("subcategories", [])
            if isinstance(subcategory, dict)
        ]
        categories.append({"id": category.get("id"), "name": category.get("name"), "subcategories": subcategories})
    return categories


def get_expense(expense_id: int) -> dict[str, object]:
    payload = _request("GET", f"get_expense/{expense_id}")
    expense = payload.get("expense", {})
    return _compact_expense(expense if isinstance(expense, dict) else {})


def _normalize_users(users: list[dict[str, object]], cost: Decimal) -> list[dict[str, object]]:
    if not users:
        raise SplitwiseError("users is required")

    paid_total = Decimal("0")
    owed_total = Decimal("0")
    normalized: list[dict[str, object]] = []
    for index, user in enumerate(users):
        if not isinstance(user, dict):
            raise SplitwiseError(f"users[{index}] must be an object")
        if "user_id" not in user and "email" not in user:
            raise SplitwiseError(f"users[{index}] must include user_id or email")
        paid = _money(user.get("paid_share", "0"), f"users[{index}].paid_share")
        owed = _money(user.get("owed_share", "0"), f"users[{index}].owed_share")
        paid_total += paid
        owed_total += owed
        normalized.append(user)

    if paid_total != cost:
        raise SplitwiseError(f"paid_share total {paid_total} must equal cost {cost}")
    if owed_total != cost:
        raise SplitwiseError(f"owed_share total {owed_total} must equal cost {cost}")
    if not any(Decimal(str(user.get("paid_share", "0"))) == cost for user in normalized):
        raise SplitwiseError(f"one user must have paid_share equal to the total cost ({cost})")
    return normalized


def create_expense(
    *,
    cost: str,
    description: str,
    group_id: int,
    users: list[dict[str, object]],
    currency: str = DEFAULT_CURRENCY,
    date: str | None = None,
    category_id: int | None = None,
    details: str | None = None,
) -> dict[str, object]:
    amount = _money(cost, "cost")
    form = {
        "cost": str(amount),
        "description": description,
        "group_id": str(group_id),
        "currency_code": currency or DEFAULT_CURRENCY,
    }

    if date:
        form["date"] = date
    if category_id is not None:
        form["category_id"] = str(category_id)
    if details:
        form["details"] = details

    for index, user in enumerate(_normalize_users(users, amount)):
        if "user_id" in user:
            form[f"users__{index}__user_id"] = str(user["user_id"])
        if "email" in user:
            form[f"users__{index}__email"] = str(user["email"])
        if "first_name" in user:
            form[f"users__{index}__first_name"] = str(user["first_name"])
        if "last_name" in user:
            form[f"users__{index}__last_name"] = str(user["last_name"])
        form[f"users__{index}__paid_share"] = str(user.get("paid_share", "0"))
        form[f"users__{index}__owed_share"] = str(user.get("owed_share", "0"))

    payload = _request("POST", "create_expense", form_data=form)
    errors = payload.get("errors", {})
    if errors:
        raise SplitwiseError("Splitwise rejected expense creation", errors=errors)

    expenses = payload.get("expenses", [])
    if isinstance(expenses, list) and expenses and isinstance(expenses[0], dict):
        return _compact_expense(expenses[0])
    return payload


def _compact_expense(expense: dict[str, object]) -> dict[str, object]:
    users = []
    for item in expense.get("users", []):
        if not isinstance(item, dict):
            continue
        user = item.get("user", {})
        if not isinstance(user, dict):
            user = {}
        users.append(
            {
                "user_id": user.get("id"),
                "name": _name(user),
                "paid_share": item.get("paid_share"),
                "owed_share": item.get("owed_share"),
                "net_balance": item.get("net_balance"),
            }
        )

    category = expense.get("category", {})
    return {
        "id": expense.get("id"),
        "description": expense.get("description"),
        "cost": expense.get("cost"),
        "currency_code": expense.get("currency_code"),
        "date": expense.get("date"),
        "group_id": expense.get("group_id"),
        "category": category.get("name") if isinstance(category, dict) else None,
        "users": users,
    }
