# Reopen AnyDesk When Closed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ensure `reopenAnyDesk: true` attempts to start AnyDesk even when no process was running before the kill phase.

**Architecture:** Keep the existing command envelope and executable discovery. Extract the post-kill reopen decision into a small Go helper so the zero-match behavior can be tested directly, then apply the same boolean rule in the Python consumer.

**Tech Stack:** Go 1.24, Python 3, GitHub Actions.

## Global Constraints

- Do not broaden the allowlisted command surface.
- Do not change the n8n payload shape.
- Preserve `reopenAttempted` and `reopened` result fields.
- A missing executable must return `reopened: false`, not reject the command.

---

### Task 1: Go regression and fix

**Files:**
- Modify: `agent-go/main_test.go`
- Modify: `agent-go/main.go`

**Interfaces:**
- Produces: `applyReopen(outcome, bool, string, func(string) bool) outcome`

- [ ] **Step 1: Write the failing test**

Add a test that passes `outcome{Matched: 0}`, requests reopening, and asserts the opener is called and `ReopenAttempted` is true.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./...`
Expected: FAIL because `applyReopen` does not exist.

- [ ] **Step 3: Write minimal implementation**

Add `applyReopen` and replace the inline `result.Matched > 0` gate in `consume`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./...`
Expected: PASS.

### Task 2: Python parity

**Files:**
- Modify: `agent/kill_anydesk_agent.pyw`

- [ ] **Step 1: Apply the same decision rule**

Set `outcome["reopenAttempted"]` directly from the validated `reopenAnyDesk` boolean and call `reopen_anydesk` whenever it is true.

- [ ] **Step 2: Verify syntax**

Run: `python -m py_compile agent/kill_anydesk_agent.pyw`
Expected: PASS.

### Task 3: Documentation and CI

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Update behavior documentation**

Document that reopening is attempted even if AnyDesk was already closed.

- [ ] **Step 2: Verify branch CI**

Confirm Go tests, vet, module verification, and Windows build pass in GitHub Actions.
