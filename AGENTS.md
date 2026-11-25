# Instructions for AI coding agents

Read [CONTRIBUTING.md](CONTRIBUTING.md) before you change code or open a pull request.

## Commands

- Use `make` targets, not raw `go` commands. Raw `go` commands do not set the required `netgo,sqlite_fts5` build tags.
  - `make test` runs the Go tests. `make test PKG=./path` runs one package.
  - `make lint`, `make format` and `make build` lint, format and build the code.
- Run `make help` to get the up-to-date list of commands

## Tests

- Write Go tests with Ginkgo/Gomega. Use one test file for each production file.
- A test must fail when the behavior it names breaks. Do not write tests that cannot fail, such as:
  - a test that checks a mock returns what the test told it to return
  - a test that computes the expected value with the same logic as the code under test
  - a test that checks fields or constants it just set
  - a test that only checks the code runs without an error
- For a bug fix, run the new test without the fix and confirm that it fails.
- If the only possible test is tautological, say so instead of writing it.

## Code comments

- Prefer self-documenting code. Add a comment only to explain *why*, never *what* the code already states.
- Keep each comment to 1 line, 2 at most.
- In a doc comment, the text after the symbol's name must say something the name does not.
- Never put PR or issue history, rejected alternatives, or benchmark numbers in a comment. Put them in the commit message or PR body.

## Do not change without maintainer agreement

- Do not change (Open)Subsonic API routes or behavior. Clients depend on them.
- Add new UI strings only to `ui/src/i18n/en.json`. Do not machine-translate the other languages.

## Commits

- Sign off every commit with `git commit --signoff` (DCO).
- Use the format `<type>(scope): <description>`, as [CONTRIBUTING.md](CONTRIBUTING.md#commit-conventions-and-dco) describes.
- Do not force-push after the PR is open.

## Opening a pull request

- Keep the PR to the agreed scope. Do not add refactors that nobody asked for.
- Write the PR description from [.github/pull_request_template.md](.github/pull_request_template.md). Keep all of its sections. Follow the instructions in the template comment. 
- Never check these two boxes:
  - "I personally read and reviewed the entire diff..."
  - "For code changes, I have the development knowledge..."

  Only the person who submits the PR can make these claims. Leave both boxes unchecked. Tell the person to review the diff and check the boxes themselves.
- Answer the "AI use" question with "Yes".
- Leave every other checklist item unchecked unless you verified it.
- Under "How to Test", list only commands that you really ran, with their real results. Say which checks failed or did not run.
- If the change makes any visible change in the UI, add before and after screenshots or a screen recording under "Screenshots / Demos". If you cannot capture them, ask the person to add them.
- Never make up an issue link or a maintainer agreement. If there is none, ask the person.
- Keep the PR text short.
