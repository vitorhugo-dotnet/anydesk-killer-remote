# Reopen AnyDesk When Closed

## Requirement

When a valid `KILL_ANYDESK` command contains `args.reopenAnyDesk: true`, both the Go and Python agents must attempt to start AnyDesk after the kill phase, even when no AnyDesk process was running when the command arrived.

## Behavior

- `reopenAnyDesk: false` or omitted: only perform the kill phase.
- `reopenAnyDesk: true`: perform the kill phase, then attempt to start AnyDesk regardless of the number of matched processes.
- `outcome.reopenAttempted` reports whether the command requested reopening.
- `outcome.reopened` reports whether an executable was successfully started.
- A missing executable remains a successful command with `reopened: false`; it is not a rejected command.

## Scope

Change only the post-kill decision in the Go and Python agents and update documentation. The n8n workflow already sends the boolean argument and requires no behavioral change.

## Testing

Add a Go regression test proving that a reopen request triggers the opener when `matched == 0`. Keep the implementation minimal and mirror the same decision rule in Python.
