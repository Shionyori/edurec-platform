## Branch and commit rules

### General
- **Automatic branch creation and commits are allowed**, but the assistant **must stop after each logical milestone** (e.g., completing a module, finishing a key feature point) and wait for explicit user confirmation before proceeding to the next milestone.
- **Do not implement a large set of features or fixes in a single commit.** Each commit must represent a single, clear logical change.
- **Workflow example**:
  1. Create a branch (e.g., `feat/xxx`) or use an existing one.
  2. Implement one functional point → commit (following the message format) → **stop immediately and request user confirmation**.
  3. After user confirms, proceed to the next functional point → commit → stop again.
  4. Repeat until the entire task is complete.

### Branch naming
- Branch names should be clear and concise. Format: `<type>/<short-description>`, e.g.:
  - `feat/frontend-login`
  - `fix/backend-timeout`
  - `docs/readme-update`

### Commit message format
- Commit messages should be clear and concise. Format: `<type>(<scope>): <subject>`, e.g.:
  - `feat(frontend): add login button`
  - `fix(backend): resolve null pointer exception`

### Pause and confirmation mechanism
- After **every commit**, the assistant **must** explicitly prompt the user: “This commit is complete. Please review and confirm whether to continue to the next step.”
- The assistant **must not** make any new code changes or commits until the user explicitly replies with a confirmation like “continue” or equivalent.
- If the user requests modifications or rollbacks, adjust accordingly before continuing.

### Granularity of commits
- Each commit should contain a complete, independently verifiable change (e.g., adding one API endpoint, fixing one clear bug, updating one UI component).
- If a feature is decomposed into multiple sub-tasks, commit after each sub‑task and wait for confirmation.