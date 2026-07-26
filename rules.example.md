# Commit message format

## Main rules

- Use the get_current_branch command to get the current branch name
- Extract the task number from the branch name (e.g., from feature/TASK-123-something extract 123)
- In the first line, specify the task number and a brief description of changes
- The task number must be strictly in FEATURE-{number} format. The prefix TASK- is always fixed
- First line format: "TASK-{number}: ServiceName. Brief description"
- Leave a blank line
- Add a link to the task (if available)
- Leave a blank line
- Then list changes as a bulleted list (-)
- Each item must start with a past tense verb (Added, Fixed, Implemented, etc.)
- Write in English
- Be concise, do not add extra text

## Example

```text
FEATURE-123: ServiceName. Add handler

https://youtrack.company.com/issue/FEATURE-123

- Added retry for webclient
- Added UserCreate handler
- Added migration for user create
```
