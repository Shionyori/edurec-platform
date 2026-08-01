# Notices
- Never try to implement a lot at once.
- Never put a ton of features or changes into a single commit.
- After a stage of work, remember to update relevant documentation and tests (if exist).

# Branch and commit rules

## General
- If needed, create a new branch (just need to give cmd advice). But don't create a new branch for every single commit. 
- After each change, recommend a commit message (if needed to commit).
- Each commit should be a single logical change. If a change is too big, break it down into smaller commits.
- Considering the rules above, if a change is too big, you may need to stop and give commit advice, after mannual review, and then continue to implement the rest of the change in a new commit.

## Branch format
- Branch names should be clear and concise.
- For example, "feat/frontend-new-feature" or "fix/backend-login-issue"...

## Commit messages format
- Commit messages should be clear and concise.
- For example, "feat(frontend): add new feature" or "fix(backend): resolve issue with login"...

# Comments format
- Comments should be clear and concise.
- Better to use chinese.
- Comments should be added at proper places (e.g., explain diffcults/key points, mark code function to make structure more clear), not everywhere.
- If a comment is in the middle of a piece of code, it should placed just after the line, don't break the line.