# Memory Command Interface Design

## Role
Senior Go Engineer designing the CLI `memory` subcommand interface for `a-flex-box/cli`. This command manages a persistent memory/knowledge store—capturing findings, hypotheses, questions, tools, ideas, and mistakes across iterations.

## Project Context
- **Framework**: Cobra (CLI)
- **Pattern**: Each subcommand group lives in `cmd/{name}/` with `NewCmd()` returning `*cobra.Command`, registered in `cmd/root.go` via `rootCmd.AddCommand(cmdmemory.NewCmd())`
- **Existing example**: `cmd/history/` — parent command + subcommands (e.g., `history add`)
- **Internal package**: `internal/memory/` (to be implemented) for the core storage engine
- **Storage**: Local JSON file (`~/.config/a-flex-box/memory.json`) per profile, indexed by `(repo, subject)`

## Directory Structure

```text
cmd/
  └── memory/
      ├── root.go        # Parent command, adds subcommands
      ├── save.go        # cli memory save
      ├── load.go        # cli memory load
      ├── list.go        # cli memory list
      ├── search.go      # cli memory search
      ├── prune.go       # cli memory prune
      └── star.go        # cli memory star
```

## Subcommand Specifications

### 1. `save` — Persist a memory entry

**Use**: `cli memory save [content] [flags]`

| Flag       | Type     | Required | Default | Description                                        |
|-----------|----------|----------|---------|----------------------------------------------------|
| `--repo`   | `string` | No       | `.`     | Repository path or identifier                      |
| `--subject`| `string` | No       | auto    | Subject/title for the memory entry                 |
| `--type`   | `string` | No       | `note`  | Category: `finding`, `hypothesis`, `question`, `tool`, `idea`, `mistake`, `note` |
| `--tags`   | `string` | No       | `""`    | Comma-separated tags                               |
| `--star`   | `bool`   | No       | `false` | Auto-star the new entry                            |

**Behavior**:
- If `content` is provided inline, capture it directly.
- If `content` is `-` or omitted, read from stdin (piped content or multi-line input).
- If `--subject` is omitted, auto-generate from first 60 chars of content.
- Validate `--type` against the allowed set; reject unknown types with a clear error.

**Output**:
```
✅ Memory saved [id: 7a3f9c] — "Fixed nil pointer in worker pool" (finding)
```

---

### 2. `load` — Retrieve a memory entry or show index

**Use**: `cli memory load [flags]`

| Flag       | Type     | Required | Default | Description                                        |
|-----------|----------|---------|---------|----------------------------------------------------|
| `--repo`   | `string` | No       | `.`     | Repository path or identifier                      |
| `--subject`| `string` | No       | `""`    | Subject to load. Omit to show repo INDEX.          |

**Behavior**:
- If `--subject` is provided: fetch the single entry, print its full content as raw markdown.
- If `--subject` is omitted: show a table INDEX of all entries for the repo (see list format).

**Output (with --subject)**:
```markdown
## Fixed nil pointer in worker pool
**Type**: finding  **Tags**: bug, worker, concurrency
  
When dispatcher.Close() is called during pending submit, 
the next submit panics on s.active == nil.
Fix: add nil guard in dispatch().

---
id: 7a3f9c | repo: cli | created: 2026-06-12 | score: 0
```

**Output (without --subject, INDEX mode)**:
```
 Memory Index for repo "cli" (5 entries)

  ID       | Subject                                     | Type      | Score | Created
 ----------|---------------------------------------------|-----------|-------|----------
  7a3f9c   | Fixed nil pointer in worker pool            | finding   | 0     | 2026-06-12
  b1e4a2   | Try zerolog for structured logging          | hypothesis| 3     | 2026-06-11
  9c8d7f   | Add config hot-reload via fsnotify          | idea      | 5     | 2026-06-10
```

---

### 3. `list` — List all entries for a repo

**Use**: `cli memory list [flags]`

| Flag       | Type     | Required | Default | Description               |
|-----------|----------|---------|---------|---------------------------|
| `--repo`   | `string` | No       | `.`     | Repository path or identifier |

**Behavior**:
- Print a human-readable table of all entries for the repo (same format as INDEX above).
- Sort by `score` descending, then `created` descending.

**Output**:
```
 Memory for repo "cli" (5 entries)

  ID       | Subject                                     | Type      | Score | Created
 ----------|---------------------------------------------|-----------|-------|----------
  9c8d7f   | Add config hot-reload via fsnotify          | idea      | 5     | 2026-06-10
  b1e4a2   | Try zerolog for structured logging          | hypothesis| 3     | 2026-06-11
  7a3f9c   | Fixed nil pointer in worker pool            | finding   | 0     | 2026-06-12
  ...
```

---

### 4. `search` — Search entries by keyword

**Use**: `cli memory search [flags]`

| Flag       | Type     | Required | Default | Description                  |
|-----------|----------|---------|---------|------------------------------|
| `--repo`   | `string` | No       | `.`     | Repository path or identifier |
| `--keyword`| `string` | Yes      | —       | Search term (fuzzy matched against subject, tags, and content) |

**Behavior**:
- Fuzzy/partial match against `subject`, `tags`, and `content` fields.
- Print results in the same table format as `list`.
- Sort by relevance score descending, then star score descending.

**Output**:
```
 Search results for "nil pointer" in repo "cli" (2 matches)

  ID       | Subject                                     | Type      | Score | Created
 ----------|---------------------------------------------|-----------|-------|----------
  7a3f9c   | Fixed nil pointer in worker pool            | finding   | 0     | 2026-06-12
  d4e5f6   | Investigate nil pointer in parser            | question  | 2     | 2026-06-09
```

---

### 5. `prune` — Delete memory entries

**Use**: `cli memory prune [flags]`

| Flag       | Type     | Required | Default | Description                  |
|-----------|----------|---------|---------|------------------------------|
| `--repo`   | `string` | No       | `.`     | Repository path or identifier |
| `--subject`| `string` | Yes      | —       | Subject or ID to delete      |

**Behavior**:
- Delete the entry matching `--subject` (matched by subject text or by ID prefix).
- Before deleting, print the entry summary and ask for confirmation (`y/N`).
- If entry not found, print a clear error.

**Output**:
```
 Memory entry "7a3f9c — Fixed nil pointer in worker pool"
 Are you sure you want to delete it? [y/N] y
 ✅ Memory entry deleted.
```

---

### 6. `star` — Rate/star an entry

**Use**: `cli memory star [flags]`

| Flag       | Type     | Required | Default | Description                  |
|-----------|----------|---------|---------|------------------------------|
| `--repo`   | `string` | No       | `.`     | Repository path or identifier |
| `--subject`| `string` | No       | `""`    | Subject to star (mutex with --id) |
| `--id`     | `string` | No       | `""`    | Entry ID to star (mutex with --subject) |
| `--score`  | `int`    | No       | `1`     | Score (0-5, where 0=unstar, 5=essential). Default increments by 1. |

**Behavior**:
- Must provide exactly one of `--subject` or `--id`.
- If `--score` is omitted: increment current score by 1 (cap at 5).
- If `--score` is provided: set score to exact value.
- Score `0` removes the star (marks as non-essential).

**Output**:
```
 ⭐ Memory entry "7a3f9c — Fixed nil pointer in worker pool"
    Score: 3 → 4
```

---

## Parent Command Registration (`cmd/memory/root.go`)

```go
package memory

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "memory",
        Short: "Persistent memory for findings, hypotheses, ideas, and more",
        Example: "cli memory save \"The config hot-reload uses fsnotify\" --type=idea --star",
    }
    cmd.AddCommand(newSaveCmd())
    cmd.AddCommand(newLoadCmd())
    cmd.AddCommand(newListCmd())
    cmd.AddCommand(newSearchCmd())
    cmd.AddCommand(newPruneCmd())
    cmd.AddCommand(newStarCmd())
    return cmd
}
```

Registered in `cmd/root.go` as:

```go
cmdmemory "github.com/A-Flex-Box/cli/cmd/memory"
// ...
rootCmd.AddCommand(cmdmemory.NewCmd())
```

## Internal Package Interface (`internal/memory/`)

The `internal/memory/` package will expose the following function signatures (to be designed in a subsequent spec):

```go
// Entry is a single memory record
type Entry struct {
    ID        string   `json:"id"`
    Repo      string   `json:"repo"`
    Subject   string   `json:"subject"`
    Type      string   `json:"type"`       // finding|hypothesis|question|tool|idea|mistake|note
    Tags      []string `json:"tags,omitempty"`
    Content   string   `json:"content"`
    Score     int      `json:"score"`      // 0-5
    Created   string   `json:"created"`    // ISO 8601
    Updated   string   `json:"updated,omitempty"`
}

func Save(repo, subject, mtype, content string, tags []string, star bool) (string, error)
func Load(repo, subject string) (*Entry, error)
func List(repo string) ([]Entry, error)
func Search(repo, keyword string) ([]Entry, error)
func Prune(repo, subject string) error
func Star(repo, subject, id string, score int) error
```

## Notes
- All commands respect `--repo` defaulting to current directory (`.`), allowing per-project memory namespacing.
- No interactive prompts except the delete confirmation in `prune`.
- Output is human-readable tables and raw markdown — no JSON output in this iteration (can be added later via a global `--json` flag).
- ID is a short hash (6 chars hex) derived from `(repo + subject + created_timestamp)`.
