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
2. Also set up [Authelia](https://www.authelia.com), locally or remotely.
3. Run the backend locally by 
    - Use `docker compose up -d db`
    - Edit your `.env` file: Change `VMAIL_DB_HOST` to point to `localhost`.
3. Set up Authelia locally
    - Follow [Authelia's docs](https://www.authelia.com/docs/getting-started/installation/) to run it locally.
    - Set your `.env` file so that it points to the local Authelia instance.
      TODO Complete this

### Tooling

The project includes several utility scripts in the `scripts/` directory. See [`scripts/README.md`](../scripts/README.md) for detailed documentation.

**Available scripts:**

- **`check.sh`** - Runs all formatting, linting, and tests. Use `./scripts/check.sh` before committing new code and ensure all checks pass locally.
- **`roadmap-burndown.go`** - Analyzes git history of `ROADMAP.md` to generate a CSV burndown chart showing task completion over time.
