# n8n Workflow Git Sync Design

## Goal

Synchronize selected n8n workflows with one or more repositories owned by `vitorhugo-dotnet`, in both directions, without automatically overwriting a concurrent edit.

## Scope

The initial implementation stores mappings and sync state in the `Workflow Git Sync` Data Table. A mapping is one workflow-to-repository-path destination; the same workflow can therefore have multiple rows and sync to multiple repositories. Each row has `workflowId`, `workflowName`, `repository`, `branch`, `filePath`, `status`, `lastN8nHash`, `lastGitHubSha`, `lastSyncedAt`, `conflictN8nHash`, `conflictGitHubSha`, `resolution`, and `lastError`.

Only repositories matching `vitorhugo-dotnet/<repository>` are accepted. The canonical Git file is a normalized JSON object containing `name`, `nodes`, `connections`, `settings`, `pinData`, and `description`; operational metadata, credentials, execution records, timestamps, and active state are excluded.

## Components

`Workflow Sync — Push to GitHub` runs every five minutes. It reads active mappings, fetches each n8n workflow and GitHub file, compares the normalized n8n SHA-256 hash and GitHub blob SHA with the stored baseline, then:

- pushes n8n to GitHub when only n8n changed;
- refreshes the baseline when neither side changed;
- marks the mapping `CONFLICT` when both sides changed;
- does not change GitHub when GitHub alone changed, leaving that direction to the pull workflow.

`Workflow Sync — Pull from GitHub` runs every five minutes after the push worker. It reads active mappings, fetches the GitHub JSON, validates its allowlisted top-level shape, compares it with the baseline, and applies it with the n8n Workflow API only when GitHub alone changed. It marks the mapping `CONFLICT` when both sides changed.

`Workflow Sync — Resolve Conflict` is a protected n8n form. It lists a mapping and requires exactly `N8N_WINS` or `GITHUB_WINS`. It performs the chosen one-way update, then stores the new hashes, clears conflict hashes, and returns the mapping to `SYNCED`.

## Conflict and notifications

No worker overwrites a `CONFLICT` mapping. On the first transition to `CONFLICT`, it records both hashes, timestamp, repository, workflow, branch, and file path, then sends one e-mail to `vitorhugoalvesferreira@gmail.com`. Repeated polling does not send duplicate alerts while the row remains in conflict.

## Credentials and activation

The workflows use the existing `GitHub account` and `Gmail account` credentials. They require one new least-privilege `n8n API` credential pointing to the same n8n instance. Its API key must be created in n8n Settings and must not be saved in Git. Workflows are created unpublished; after the credential is selected and the initial mapping is reviewed, they may be published.

## Non-goals

This version does not provision GitHub webhooks, synchronize credentials, create workflows absent from n8n, or update the active/published state from Git. Polling is intentional: it supports multiple repositories without placing webhook signing secrets in a Data Table.
