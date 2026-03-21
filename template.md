---
id: {{ID}}
title: {{Short imperative description}}
status: [To Do|In Progress|To Merge|Abandoned|Done]
priority: {{int}}
effort: [Unknown|Trivial|Small|Medium|Large]
assignee: [claude|human]
created_date: YYYY-MM-DD
labels: [bugfix|enhancement|feature, core]
swimlane: [Core Library|Tooling/Backflow|Tooling/Claude Environment|Publication|Core Documentation]
source_file: {{file}}:{{line}}
---

<!-- Ticket Quality Guidelines:
- Include exact file paths and line numbers for code to modify
- Quote relevant function bodies or struct definitions inline
- Show before/after code blocks where applicable
- Reference data flow: which functions call what, what structs feed into what
- The ticket IS the prompt — precision saves ~30% of agent token costs
-->

## Summary

One to three sentences describing what needs to change and why.

## Current State

Describe the relevant code as it exists today. Include line numbers and
code snippets where helpful.


## Analysis & Recommendations

## TDD Plan

### RED

```cpp
TEST_CASE("Description of expected behavior") {
    // Setup and assertions
}
```

### GREEN

1. First implementation step
2. Second implementation step

### REFACTOR

- Cleanup or follow-up improvements

<!-- ## Results

### Complexity Metrics

Run `./scripts/backlog.py complexity-summary` and paste output here.

-->

<!-- ## Attachments

List attached resources if this ticket has a .attachments/ directory.
Remove this section if there are no attachments.

Example:
- `style-lab-results.json` — Converged design tokens from 542 voting rounds
- `screenshot-broken-layout.png` — Visual evidence of the rendering bug

-->
