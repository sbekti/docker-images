---
name: splitwise
description: Use Splitwise to inspect or create shared expenses.
metadata:
  requireAuth: true
---

# Splitwise

Use the `mcp_hermes_tools_splitwise_*` tools exposed by the local Hermes tools
MCP server.

## Workflow

1. Use the read-only group, friend, category, and expense tools to discover
   identifiers instead of guessing them.
2. Clarify the group, payer, participants, amounts, currency, date, category,
   and description when the request does not provide them.
3. For every participant, provide either `user_id` or `email` plus explicit
   `paid_share` and `owed_share` values.
4. Require paid shares and owed shares to each sum exactly to the expense cost.
5. Show the final expense details and obtain explicit confirmation before any
   create call.
6. Call `splitwise_create_expense` with the selected `group_id`, complete
   `users` list, and `confirmed: true` only after that confirmation.
7. Check the returned payload for errors before reporting success.

Never request, read, print, or expose Splitwise credentials. Treat
authentication as a built-in capability.

For local operator testing, use:

```bash
python /opt/hermes-tools/skills/splitwise/splitwise_cli.py groups
```
