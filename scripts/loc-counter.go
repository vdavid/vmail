package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type fileStats struct {
	total    int
	ts       int
	goTotal  int
	goProd   int
	goTest   int
	tsProd   int
	tsTest   int
	docs     int
	other    int
	comments []string
}

func main() {
	// Get all commits on the main branch
	commits, err := getCommits()
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error getting commits: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	// Group commits by date and get the latest commit per day
	dailyCommits := groupCommitsByDate(commits)

	// Get dates sorted
	dates := make([]string, 0, len(dailyCommits))
	for date := range dailyCommits {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Output CSV header
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()
	err = writer.Write([]string{"date", "total", "ts", "go", "go prod", "go test", "ts prod", "ts test", "docs", "other", "comments"})
	if err != nil {
		return
	}

	// Process each date
	for _, date := range dates {
		commit := dailyCommits[date]
		stats, err := countLinesForCommit(commit.hash, commit.messages)
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "Error counting lines for commit %s: %v\n", commit.hash, err)
			if err != nil {
				return
			}
			continue
		}

		// Format comments (first line of each commit message, joined with semicolon)
		comments := strings.Join(stats.comments, "; ")

		// Write CSV row
		err = writer.Write([]string{
			date,
			fmt.Sprintf("%d", stats.total),
			fmt.Sprintf("%d", stats.ts),
			fmt.Sprintf("%d", stats.goTotal),
			fmt.Sprintf("%d", stats.goProd),
			fmt.Sprintf("%d", stats.goTest),
			fmt.Sprintf("%d", stats.tsProd),
			fmt.Sprintf("%d", stats.tsTest),
			fmt.Sprintf("%d", stats.docs),
			fmt.Sprintf("%d", stats.other),
			comments,
		})
		if err != nil {
			return
		}
	}
}

type commit struct {
	hash     string
	date     string
	messages []string
}

func getCommits() ([]commit, error) {
	cmd := exec.Command("git", "log", "--format=%H|%cd|%s", "--date=short", "main")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %w", err)
	}

	var commits []commit
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}

		commits = append(commits, commit{
			hash:     parts[0],
			date:     parts[1],
			messages: []string{parts[2]},
		})
	}

	return commits, scanner.Err()
}

func groupCommitsByDate(commits []commit) map[string]commit {
	dailyCommits := make(map[string]commit)

	for _, c := range commits {
		existing, ok := dailyCommits[c.date]
		if !ok {
			// First commit for this date (latest chronologically since git log is reverse order)
			dailyCommits[c.date] = c
		} else {
			// Multiple commits on the same day - keep the latest commit hash (first one we saw)
			// But collect all messages from all commits on that day
			existing.messages = append(existing.messages, c.messages...)
			dailyCommits[c.date] = existing
		}
	}

	return dailyCommits
}

func countLinesForCommit(commitHash string, messages []string) (*fileStats, error) {
	stats := &fileStats{
		comments: messages,
	}

	// Get all files at this commit
	files, err := getFilesAtCommit(commitHash)
	if err != nil {
		return nil, err
	}

	// Count lines for each file
	for _, file := range files {
		// Skip pnpm-lock.yaml (generated lockfile)
		if strings.HasSuffix(file, "pnpm-lock.yaml") {
			continue
		}

		lines, err := countFileLines(commitHash, file)
		if err != nil {
			// Skip files that can't be read (might be binary or deleted)
			continue
		}

		stats.total += lines

		// Categorize the file
		ext := strings.ToLower(filepath.Ext(file))
		base := filepath.Base(file)

		switch {
		case strings.HasSuffix(base, "_test.go"):
			stats.goTest += lines
			stats.goTotal += lines
		case ext == ".go":
			stats.goProd += lines
			stats.goTotal += lines
		case strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, ".test.tsx"):
			stats.tsTest += lines
			stats.ts += lines
		case ext == ".ts" || ext == ".tsx":
			stats.tsProd += lines
			stats.ts += lines
		case ext == ".md" || base == "LICENSE":
			stats.docs += lines
		default:
			stats.other += lines
		}
	}

	return stats, nil
}

func getFilesAtCommit(commitHash string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", commitHash)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git ls-tree: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file != "" {
			files = append(files, file)
		}
	}

	return files, scanner.Err()
}

func countFileLines(commitHash, filepath string) (int, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commitHash, filepath))
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get file content: %w", err)
	}

	// Count lines (including blank lines)
	lines := strings.Count(string(output), "\n")
	// If the file doesn't end with a newline, count the last line
	if len(output) > 0 && !strings.HasSuffix(string(output), "\n") {
		lines++
	}

	return lines, nil
}
