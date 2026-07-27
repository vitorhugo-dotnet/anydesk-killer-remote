# n8n Workflow Git Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide safe, multi-repository bidirectional synchronization between selected n8n workflows and GitHub JSON files.

**Architecture:** Store mapping and baseline state in one n8n Data Table. Separate scheduled push and pull workers compare each side to the baseline, while a protected resolver handles conflicts explicitly.

**Tech Stack:** n8n Data Tables, n8n Workflow SDK, GitHub node, Gmail node, SHA-256 in Code nodes.

## Global Constraints

- Accept only repositories owned by `vitorhugo-dotnet`.
- Export only normalized workflow definitions; never credentials, execution data, or active state.
- Both-side changes create `CONFLICT` and send one e-mail; neither side is overwritten.
- Workflows remain unpublished until the n8n API credential is configured.

---

### Task 1: Create mapping state

**Files:**
- Create: `n8n/Workflow Git Sync - Push.json`
- Create: `n8n/Workflow Git Sync - Pull.json`
- Create: `n8n/Workflow Git Sync - Resolve Conflict.json`

- [ ] Create `Workflow Git Sync` Data Table with mapping, baseline, conflict, and error columns.
- [ ] Seed the Kill AnyDesk V2 mapping for `vitorhugo-dotnet/anydesk-killer-remote`, `main`, `n8n/Kill AnyDesk V2.json`.
- [ ] Verify the table schema and row using the n8n Data Table API.

### Task 2: Create push worker

**Files:**
- Create: `n8n/Workflow Git Sync - Push.json`

- [ ] Build a five-minute scheduled worker that reads enabled, non-conflicted mappings.
- [ ] Normalize an n8n workflow, compare its SHA-256 with the baseline and GitHub blob SHA, and create or update only when n8n alone changed.
- [ ] Mark simultaneous changes as `CONFLICT` and e-mail the configured recipient once.
- [ ] Validate the workflow before creation and leave it unpublished.

### Task 3: Create pull worker

**Files:**
- Create: `n8n/Workflow Git Sync - Pull.json`

- [ ] Build a five-minute scheduled worker that reads GitHub JSON for each enabled mapping.
- [ ] Validate the allowlisted workflow shape and update n8n only when GitHub alone changed.
- [ ] Persist the new baseline or register a conflict without overwriting either side.
- [ ] Validate the workflow before creation and leave it unpublished.

### Task 4: Create conflict resolver and documentation

**Files:**
- Create: `n8n/Workflow Git Sync - Resolve Conflict.json`
- Modify: `README.md`

- [ ] Build a protected form that resolves one row with `N8N_WINS` or `GITHUB_WINS`.
- [ ] Persist the selected winner and clear the conflict only after the chosen side succeeds.
- [ ] Document the credential setup, mapping fields, and activation order.
- [ ] Export all workflow JSON files to the repository.

### Task 5: Verify and publish source

**Files:**
- Modify: `n8n/*.json`

- [ ] Validate each n8n workflow with the n8n validation API.
- [ ] Verify Data Table schema and mapping row.
- [ ] Commit and push the exports and documentation to `main`.
