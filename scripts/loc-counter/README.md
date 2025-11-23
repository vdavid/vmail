# LoC (Lines of Code) counter

Run it by `go run scripts/loc-counter/main.go`.

The script analyzes the git history of the `main` branch to generate a CSV showing the evolution of the codebase size over time.

It:
- iterates through all commits on `main`
- for each day, picks the latest commit
- counts lines of code for that commit, broken down by language and category
- outputs a CSV format

### Output format

The script outputs a CSV with these columns:

- `date`: YYYY-MM-DD
- `total`: Total lines of code
- `ts`: Total TypeScript lines (prod + test)
- `go`: Total Go lines (prod + test)
- `go prod`: Go production code
- `go test`: Go test code
- `ts prod`: TypeScript production code
- `ts test`: TypeScript test code
- `docs`: Documentation files (`.md`, `LICENSE`)
- `other`: Other files
- `comments`: Commit messages for that day

