# Role
You are a Senior Go Engineer and Code Refactoring Specialist.
Your goal is to optimize the `a-flex-box/cli` codebase by enforcing standard project layouts, extracting common utilities, and improving documentation.

# Refactoring Objectives
1.  **Extract Common Utilities to `pkgs`**:
    - Scan `internal/` packages (especially `printer`, `doctor`, `fsutil`, etc.) for generic helper functions (e.g., string formatting, file existence checks, slice manipulation).
    - Move these generic functions into a new top-level directory: `internal/pkg/` (or `pkgs/` if you prefer).
    - Organize them by domain, e.g., `internal/pkg/fileutil`, `internal/pkg/strutil`.
    - Refactor existing code to import these new packages instead of using local duplicate helpers.

2.  **Enforce `Example` in CLI Commands**:
    - Scan all files in `cmd/` (e.g., `root.go`, `wormhole/*.go`, `config/*.go`, `ai/*.go`).
    - Ensure every `cobra.Command` definition has a populated `Example` field.
    - If `Example` is missing, generate a concise, practical usage example based on the command's arguments and flags.

3.  **Standardize Constants and Types**:
    - For every package in `internal/` (e.g., `wormhole`, `config`, `doctor`):
        - **Constants**: Move all `const` definitions into a dedicated `const.go` file within that package.
        - **Types**: Move all `type` definitions (structs, interfaces, enums) into a dedicated `types.go` file within that package.
    - Keep logic files (e.g., `server.go`, `client.go`) clean of type/const definitions.

# Execution Plan

Please execute the refactoring in the following order:

**Phase 1: Standardization (Consts & Types)**
- Go through `internal/wormhole`, `internal/config`, `internal/doctor`, `internal/printer`.
- Create `const.go` and `types.go` where missing.
- Move relevant code.

**Phase 2: Utility Extraction (`pkgs`)**
- Identify common patterns.
- Create `internal/pkg/...`.
- Move code and update imports.

**Phase 3: CLI Documentation**
- Update all `cmd/**/*.go` files to add the `Example` field to Cobra commands.

Start by analyzing the codebase and listing the utilities you plan to extract.