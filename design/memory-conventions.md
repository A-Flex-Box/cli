# Memory File Conventions

> Storage path: `~/.claude/memory/<repo_name>/`

## Overview

Each repository gets a flat directory of markdown files under `~/.claude/memory/<repo_name>/`. These files serve as persistent, human-readable knowledge bases — indexed by a single `INDEX.md` and organized per-subject.

---

## INDEX.md

A single table at `~/.claude/memory/<repo_name>/INDEX.md` catalogs all subject memory files.

### Columns

| Column | Description |
|--------|-------------|
| **subject** | Short topic name, same as the filename stem |
| **keywords** | Comma-separated search tags |
| **entries** | Number of discrete entries across all sections |
| **top_attention** | Current attention score (0.0–1.0) |
| **last_accessed** | ISO 8601 date of last read/write |
| **summary** | One-line topic summary |

### Example Row

```
| auth | jwt, oauth, session, clerk | 12 | 0.87 | 2026-06-12 | Authentication flows and token management |
```

---

## Subject Memory File: `<subject>.md`

Each subject file lives at `~/.claude/memory/<repo_name>/<subject>.md` and uses the following template:

```markdown
# <Subject>

## Findings

- [Date] Confirmed finding with citation/source

## Hypotheses

- [Date] Hypothesis description — status: untested / confirmed / rejected

## Open Questions

- [Date] Question that remains unanswered

## Tools & Techniques

- [Date] Tool or technique learned, with brief usage note

## Ideas

- [Date] Idea or proposal for future consideration

## Mistakes

- [Date] Mistake made, impact, and lesson learned
```

Each entry should be prefixed with an ISO 8601 date (`YYYY-MM-DD`) to support chronological ordering.

---

## Trash File Convention

Deleted or archived entries are moved to:

```
~/.claude/memory/<repo_name>/<subject>-trash.md
```

The trash file follows the same section structure as the live file. When an entry is removed from the main file, it is prepended to the corresponding section in `<subject>-trash.md` with a `[Trashed YYYY-MM-DD]` note.

---

## Attention Score Formula

The **top_attention** column in INDEX.md is computed as:

```
score = load_freq * 0.3 + recency * 0.3 + starred * 0.2 + ai_weight * 0.2
```

| Variable | Range | Description |
|----------|-------|-------------|
| `load_freq` | 0.0–1.0 | Normalized frequency this file is loaded per session |
| `recency` | 0.0–1.0 | Normalized recency of last access (1.0 = accessed today) |
| `starred` | 0.0 or 1.0 | Whether a human has manually pinned this subject |
| `ai_weight` | 0.0–1.0 | AI-assigned relevance weight based on session context |

Scores are recalculated on each session end and persisted in INDEX.md.

---

## File Lifecycle

1. **Discovery**: When a new topic arises during a session, a new `<subject>.md` is created with an empty template.
2. **Append-only**: New entries are prepended to the relevant section (most recent first).
3. **Trash**: When an entry is stale or wrong, move it to `<subject>-trash.md` rather than deleting outright.
4. **Deletion**: If a subject is no longer relevant, move `<subject>.md` → `<subject>-trash.md` and remove the INDEX.md row.

---

## Conventions

- Filenames: lowercase, kebab-case (e.g., `api-gateway.md`, `db-migrations.md`)
- Dates: always ISO 8601 (`YYYY-MM-DD`)
- One entry per bullet
- No JSON/YAML front matter — pure markdown
