# Roadmap burndown chart generator tool

Run it by `go run scripts/roadmap-burndown/main.go`.

The script analyzes the git history of `ROADMAP.md` to generate a CSV burndown chart showing task completion over time.

It:
- takes all commits where `ROADMAP.md` changed
- counts total and completed tasks (supports both `* [ ]` and `- [ ]` checkbox formats)
- aggregates the data by day (uses the latest commit each day)
- fills empty days using the previous day's data
- outputs a CSV format ready to copy-paste [here](https://docs.google.com/spreadsheets/d/1uy7wZSESecJlwaK9AzEotQIzlOYLvi84jPqQVcSeWIo/edit?gid=0#gid=0)

### Output format

The script outputs a CSV with these columns:

- `date` - In YYYY-MM-DD format
- `total_tasks` - Total number of tasks (checked and unchecked)
- `done_tasks` - Number of completed tasks (checked)
- `commit_message` - The first line of the last commit message of the day

