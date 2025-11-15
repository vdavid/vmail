# Contributing

Thanks for your interest in contributing to V-Mail! You're most welcome to do so.
The easiest way to contribute is to fork the repo, make your changes, and submit a PR.
This doc is here to help you get started.

(This doc is a WIP. If you have questions, please .)

## Links

- [README.md](README.md) is the main README for the project with install instructions and other user-level info.
- [CONTRIBUTING.md](CONTRIBUTING.md) is the file you're reading right now.
- [docs/architecture.md](docs/architecture.md) is the docs for technical decisions, high-level overview, and the such.
- [docs/features.md](docs/features.md) describes that V-Mail can do.
- [docs/style-guide](docs/style-guide.md) is **the style guide**. Make sure to read it and re-read it periodically.
- [docs/testing.md](docs/testing.md) tells you how to test.
- [scripts/README.md](scripts/README.md) contains docs for the additional scripts.

## Development getting started

It's a bit different from the ["Running"](README.md#running) section of the main README.

This setup lets you run the Go backend and the React frontend locally for debugging.

1. Run a Postgres v14+ instance and make it available on some port, either on your localhost or elsewhere.
    - Edit your `.env` file: Change `VMAIL_DB_HOST` and any others needed to point to your Postgres instance.
2. Also set up [Authelia](https://www.authelia.com), locally or remotely.
    - Follow [Authelia's docs](https://www.authelia.com/docs/getting-started/installation/) to run it locally.
    - Set your `.env` file so that it points to the local Authelia instance.
3. TODO Continue...

### Tooling

The project includes several utility scripts in the `scripts/` directory. See [`scripts/README.md`](../scripts/README.md) for detailed documentation.

## Testing
`scripts/check.sh`uns all formatting, linting, and tests.
Always use `./scripts/check.sh` before committing new code and ensure all checks pass locally.

More ideas to make it efficient:

```bash
./scripts/check.sh # Run all checks (backend and frontend)
./scripts/check.sh --backend # Run only backend checks
./scripts/check.sh --frontend # Run only frontend checks
./scripts/check.sh --check <check-name> # Run a specific check
./scripts/check.sh --help # Show help, including a list of available checks
```

### Dev process

Always follow this process when developing in this project:

1. Before developing a feature, make sure to do the planning and know exactly what you want to achieve.
2. Do the changes, in small iterations. Adhere to the [style guide](docs/style-guide.md)!
3. Use `./scripts/check.sh` to check that everything is still working.
    - Or use a subset, for example, if you only touch the front end.
    - Even fix gocyclo's cyclomatic complexity warnings! I know it's a pain, but it's helpful to keep Go funcs simple.
4. Make sure to add tests for the new code. Think about unit tests, integration tests, and end-to-end tests.
5. Update any related docs.
6. Before you call it done, check out the diff of your changes (use `git diff`) and make sure everything is actually
  needed. Revert unneeded changes.
7. Rerun `./scripts/check.sh` to make sure everything still works.
8. Suggest a commit message, in the format seen in the style guide. 