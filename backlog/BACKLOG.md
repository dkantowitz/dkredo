# Backlog System Operating Conventions

This document describes the operating conventions for the project backlog system.

## Ticket Lifecycle

Tickets progress through the following states:

1. **To Do** — Ticket is ready to be worked on
2. **In Progress** — Work has started, changes are being made
3. **To Merge** — Work is complete, ready for review and merge
4. **Done** — Ticket has been merged to main and archived
5. **Abandoned** — Ticket is no longer relevant or has been superseded

Special states (use sparingly):
- **Blocked** — Waiting on external dependency or another ticket
- **On Hold** — Intentionally paused, not abandoned

## File Naming Convention

Ticket files follow the pattern: `{id}-{slug}.md`

- `{id}`: Zero-padded 3-digit number (001, 002, ..., 099, 100, ...)
- `{slug}`: Kebab-case summary derived from the title
- Extension: Always `.md`

Examples:
- `001-add-user-authentication.md`
- `042-fix-timezone-handling.md`

## Attachments Convention

Tickets can have an attachments directory for storing related resources (review feedback JSON, screenshots, data files, etc.).

### Directory Naming

Attachments live in a directory named after the ticket file with `.md` replaced by `.attachments`:

```
backlog/042-fix-timezone-handling.md
backlog/042-fix-timezone-handling.attachments/
    screenshot-broken-layout.png
    test-data.json
```

This keeps the ticket and its attachments visually grouped when sorted alphabetically.

### Attachments Section

If a ticket has attachments, include an `## Attachments` section in the ticket `.md` file describing what's in the directory:

```markdown
## Attachments

- `test-data.json` — Sample input that triggers the bug
- `screenshot-broken-layout.png` — Visual evidence of the rendering bug
```

If there are no attachments, omit this section entirely.

### Archive Behavior

When archiving a ticket with attachments:

1. The `.md` file moves to `backlog/archive/{name}.md`
2. The `.attachments/` directory is zipped to `backlog/archive/{name}.attachments.zip`
3. The original `.attachments/` directory is removed

This keeps the archive directory flat and reduces disk usage for historical tickets.

### Worktree Behavior

When copying a ticket to a worktree (via `dobacklog` execute mode):

- The `.md` file is copied
- If `.attachments/` exists, it's copied alongside the `.md` file

## Template Usage

New tickets start from `backlog/template.md`. The template includes:

- Standard YAML frontmatter structure
- Placeholder sections for Summary, Current State, Analysis, TDD Plan
- Commented-out Attachments section (uncomment if needed)
- Ticket quality guidelines

To create a new ticket:

```bash
./scripts/backlog.py create "Short description of the task"
```

## Archive Directory

Completed or abandoned tickets move to `backlog/archive/`:

- Ticket files: `backlog/archive/{id}-{slug}.md`
- Attachments (if any): `backlog/archive/{id}-{slug}.attachments.zip`

To archive a ticket:

```bash
./scripts/backlog.py archive {id}
```

## Frontmatter Fields

All tickets include YAML frontmatter with these fields:

### Required Fields

- **id**: Numeric ticket ID (3-digit zero-padded)
- **title**: Short imperative description of the task
- **status**: Current lifecycle state (see Status Normalization below)
- **priority**: Integer 1-4 (see Priority Scale below)
- **effort**: Estimated size (see Effort Scale below)
- **assignee**: Who's working on it (`claude` or `human`)
- **created_date**: Date ticket was created (YYYY-MM-DD format)
- **swimlane**: Which area of the project (see Swimlanes below)

### Optional Fields

- **labels**: Array of tags (`[bugfix, core]`, `[enhancement, tooling]`, etc.)
- **dependencies**: Array of ticket IDs this ticket depends on (`[003, 012]`)
- **source_file**: Where the issue was found (`src/main.py:142`)
- **related_tickets**: Other tickets related to this one
- **blocked_by**: What's preventing progress
- **target_version**: Which release this is planned for

## Status Normalization

The canonical status values are:

- `To Do`
- `In Progress`
- `To Merge`
- `Abandoned`
- `Done`
- `Blocked` (use sparingly)
- `On Hold` (use sparingly)

Status values are case-sensitive. Use these exact strings for consistency.

Some tooling accepts shortcuts that normalize to canonical values:
- `todo` → `To Do`
- `wip` / `doing` → `In Progress`
- `ready` / `review` → `To Merge`
- `complete` / `merged` → `Done`

## Swimlanes

<!-- Customize these swimlane values for your project -->

Valid swimlane values:

- **Core** — Main application or library code
- **Tooling** — Development workflow scripts and automation
- **Infrastructure** — CI/CD, deployment, environment setup
- **Documentation** — README, guides, API docs
- **Testing** — Test framework, coverage, test utilities

## Priority Scale

- **P1** — Urgent, blocks other work or critical bug
- **P2** — High priority, should be done soon
- **P3** — Normal priority, do when convenient
- **P4** — Nice to have, low urgency

Priority is an integer field: use `1`, `2`, `3`, or `4`.

## Effort Scale

Estimated size of the work:

- **Trivial** — < 30 minutes, obvious change
- **Small** — 1-2 hours, single focused change
- **Medium** — Half day, multiple related changes
- **Large** — Full day or more, complex or multi-part work
- **Unknown** — Need investigation before estimating

## Special Files

These files in `backlog/` are not tickets:

- **`template.md`** — Starting point for new tickets
- **`BACKLOG.md`** — This document (operating conventions)
- **`CLAUDE.md`** — Claude-specific guidance for backlog operations

## CLI Reference

### `scripts/backlog.py`

```bash
./scripts/backlog.py next                  # Show next ticket to work on
./scripts/backlog.py create "Description"  # Create new ticket
./scripts/backlog.py list [status]         # List tickets (optionally filter by status)
./scripts/backlog.py status {id}           # Show ticket details
./scripts/backlog.py archive {id}          # Archive completed ticket
./scripts/backlog.py merge {id}            # Merge task branch into main and clean up
./scripts/backlog.py worktree {id}         # Create worktree for ticket
./scripts/backlog.py attach {id} file      # Copy file into ticket's .attachments/ directory
./scripts/backlog.py attach {id} --stdin --name out.json  # Attach from stdin
./scripts/backlog.py query                 # Filtered ticket listing with selectable fields
./scripts/backlog.py query --status all --priority 2 --format json  # Filter + JSON output
./scripts/backlog.py contents {id}         # Dump full content of one or more tickets
./scripts/backlog.py contents 003 005 012  # Dump multiple tickets at once
./scripts/backlog.py minor add --swimlane "Tooling" "description"  # Add minor issue
./scripts/backlog.py minor list            # List open minor issues across all swimlanes
./scripts/backlog.py minor done M-TL-001   # Archive a resolved minor issue
./scripts/backlog.py minor promote M-TL-001  # Promote minor issue to a full ticket
./scripts/backlog.py complexity-summary    # Print complexity metrics as markdown
./scripts/backlog.py help                  # Print full markdown manual (all commands)
./scripts/backlog.py help query            # Print help for a single command
./scripts/backlog.py test                  # Run embedded test suite
```

## Updating Tickets During Execution

When working a ticket, **update the ticket file itself** to reflect progress and outcomes:

### Starting Work

1. Set `status: In Progress` in frontmatter
2. Note any deviations from the plan in the Analysis section

### Completing Work

1. Set `status: To Merge` (or `Done` if no review needed) in frontmatter
2. Uncomment and fill in the `## Results` section:
   - Paste complexity metrics if applicable
   - Note any deviations from the TDD plan (e.g., tests modified during REFACTOR)
   - List files created or modified
3. If the RED/GREEN/REFACTOR steps changed during implementation, update the TDD Plan to reflect what actually happened (append "**Actual:**" notes, don't delete the original plan)

### Abandoning Work

1. Set `status: Abandoned` in frontmatter
2. Add a brief note in the Results section explaining why

**The ticket is the record of what was planned AND what actually happened.** Future agents reading the ticket should be able to understand both the intent and the outcome.

## Tips for Writing Good Tickets

From the template quality guidelines:

- **Include exact file paths and line numbers** for code to modify
- **Quote relevant function bodies or struct definitions** inline
- **Show before/after code blocks** where applicable
- **Reference data flow**: which functions call what, what structs feed into what
- **Include Test Plan**: TDD R/G/R and Final Verification Checklist
- **The ticket IS the prompt** — precision saves ~30% of agent token costs

Good tickets make work faster and reduce back-and-forth clarification.
