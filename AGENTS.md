# Project overview

This project is V-Mail.
V-Mail is a self-hosted, web-based email client designed for personal use.
It uses the layout and keyboard shortcuts of Gmail to make it immediately familiar for ex-Gmail users.
It connects to an IMAP server and provides the web UI to read and send email.

## Non-goals

Compared to Gmail, this project does **not** include:

* Client-side email filters. The user should set these up on the server, typically via [Sieve](http://sieve.info/).
* A visual query builder for the search box. A simple text field is fine.
* A multi-language UI. The UI is English-only.
* 95% of Gmail's settings. V-Mail has some basic settings like emails per page and undo send delay, but that's it.
* Automatic categorization such as primary/social/promotions.
* The ability to collapse the left sidebar.
* Signature management.
* Smiley/emoji reactions to emails. This is Google's proprietary thing.

## Tech stack

V-Mail uses a **Postgres** database, a **Go** back end, a **REST** API, and a **React** front end with **TypeScript**.
V-Mail needs a separate, self-hosted [Authelia](https://www.authelia.com) instance for authentication.

# AI Context & rules

## AI dev process

### Dev process

Always follow this process when developing in this project:

1. Before developing a feature, make sure to do the planning and know exactly what you want to achieve and have a task list.
2. Before touching code of a specific domain, read the related docs within `/docs` or similar.
   List folders if you need to. Example: `docs/backend/auth.md` for auth logic, `backend/migrations/*.sql` for DB schema.
3. Do the changes, in small iterations. Adhere to the [style guide](docs/style-guide.md)!
4. Use `./scripts/check.sh` to check that everything is still working.
    - Or use a subset, for example, if you only touch the front end.
    - Even fix gocyclo's cyclomatic complexity warnings! I know it's a pain, but it's helpful to keep Go funcs simple.
5. Make sure to add tests for the new code. Think about unit tests, integration tests, and end-to-end tests.
6. Check if you added new patterns, external dependencies or architectural changes. Update all related docs.
7. Also consider updating `AGENTS.md` (incl. this very process) to keep the next agent's work streamlined.
8. Before you call it done, see the diff of your changes (e.g. with `git diff`) and make sure all changes are actually
   needed. Revert unneeded changes.
9. Rerun `./scripts/check.sh` to make sure everything still works.
10. Suggest a commit message, in the format seen in the style guide.

Always keep the dev process and style guide in mind.

## High-level map

- **Frontend**: React 19 + Vite. Entry: `frontend/src/main.tsx`. State: Zustand.
- **Backend**: Go 1.23 (Standard lib HTTP). Entry: `backend/cmd/server`.
- **Architecture**: Handlers (`api/`) -> Service/core logic (`internal/`) -> DB (`db/`).
- **Testing**: Playwright (E2E), Vitest (unit), Go `testing` package.

## Tooling

- `scripts/check.sh` is the primary quality control tool. It runs linting, formatting, and back/front end+E2E tests.
`./scripts/check.sh --frontend`, `./scripts/check.sh --backend` are options, and so is
`./scripts/check.sh --check <check-name>` (use `./scripts/check.sh --help` to list them).
- Front end: `pnpm lint`, `pnpm lint:fix`, `pnpm format`, `pnpm test`, and `pnpm test:e2e`.
- `pnpm exec playwright test --config=../playwright.config.ts --grep "{test-name}" is also helpful
for context-efficient re-running of E2E-tests.
- `migrate up` (using golang-migrate)

## "Red line" rules (do not break)
- Always make check.sh happy, including warnings and gocyclo complexity!
- Modular architecture, no global state, no package cycles.
- Sentence case titles and labels
- If you add a new tool, script, or architectural pattern, you MUST update this file (`AGENTS.md`) and any relevant docs
  before finishing your response.

## Style guide highlights

### General
- **Commits**: First line max 50 chars. Blank line. Detailed description, usually as concise bullets.
- **Writing**: Friendly, active voice, and ALWAYS sentence case titles and labels!
- **Complexity**: Max cyclomatic complexity of 15 per function. (checked by gocyclo)

### Frontend (TypeScript/React)
- **No classes**: Use modules and functional components only.
- **Strict types**: `no-explicit-any` is enforced.
- **Imports**: Organized by groups (builtin, external, internal, parent, sibling, index).
- **Formatting**: Prettier is the authority.
- **React**: Hooks rules enforced, no dangerous HTML.

### Backend (Go)
- **Comments**: Meaningful comments for public functions/types.
- **DB**: Plural table names, singular columns. Add comments in SQL and Go structs.
