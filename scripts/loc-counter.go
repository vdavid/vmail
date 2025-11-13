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
	"time"
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

	// Get all dates from first commit to last commit
	if len(dailyCommits) == 0 {
		return
	}

	// Find first and last dates
	allDates := make([]string, 0, len(dailyCommits))
	for date := range dailyCommits {
		allDates = append(allDates, date)
	}
	sort.Strings(allDates)
	firstDate := allDates[0]
	lastDate := allDates[len(allDates)-1]

	// Generate all consecutive dates
	allConsecutiveDates := generateConsecutiveDates(firstDate, lastDate)

	// Output CSV header
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()
	err = writer.Write([]string{"date", "total", "ts", "go", "go prod", "go test", "ts prod", "ts test", "docs", "other", "comments"})
	if err != nil {
		return
	}

	// Track previous stats for filling gaps
	var prevStats *fileStats

	// Process each date
	for _, date := range allConsecutiveDates {
		commit, hasCommit := dailyCommits[date]
		var stats *fileStats
		var comments string

		if hasCommit {
			// Count lines for this commit
			var err error
			stats, err = countLinesForCommit(commit.hash, commit.messages)
			if err != nil {
				_, err := fmt.Fprintf(os.Stderr, "Error counting lines for commit %s: %v\n", commit.hash, err)
				if err != nil {
					return
				}
				// Use previous stats if counting fails
				if prevStats != nil {
					// Create a copy of previous stats
					stats = &fileStats{
						total:   prevStats.total,
						ts:      prevStats.ts,
						goTotal: prevStats.goTotal,
						goProd:  prevStats.goProd,
						goTest:  prevStats.goTest,
						tsProd:  prevStats.tsProd,
						tsTest:  prevStats.tsTest,
						docs:    prevStats.docs,
						other:   prevStats.other,
					}
					comments = "-"
				} else {
					continue
				}
			} else {
				// Format comments (first line of each commit message, joined with semicolon)
				comments = strings.Join(stats.comments, "; ")
				prevStats = stats
			}
		} else {
			// No commit on this day - use previous day's stats
			if prevStats == nil {
				// No previous stats available, skip this day
				continue
			}
			// Create a copy of previous stats
			stats = &fileStats{
				total:   prevStats.total,
				ts:      prevStats.ts,
				goTotal: prevStats.goTotal,
				goProd:  prevStats.goProd,
				goTest:  prevStats.goTest,
				tsProd:  prevStats.tsProd,
				tsTest:  prevStats.tsTest,
				docs:    prevStats.docs,
				other:   prevStats.other,
			}
			comments = "-"
		}

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

		// Check path-based categorization first
		switch {
		case isGoTestPath(file):
			stats.goTest += lines
			stats.goTotal += lines
		case isTSTestPath(file):
			stats.tsTest += lines
			stats.ts += lines
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

// generateConsecutiveDates generates all dates from firstDate to lastDate (inclusive)
func generateConsecutiveDates(firstDate, lastDate string) []string {
	start, err := time.Parse("2006-01-02", firstDate)
	if err != nil {
		return []string{firstDate}
	}
	end, err := time.Parse("2006-01-02", lastDate)
	if err != nil {
		return []string{lastDate}
	}

	var dates []string
	current := start
	for !current.After(end) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1)
	}
	return dates
}

// isGoTestPath checks if a file path should be categorized as Go test code
func isGoTestPath(file string) bool {
	return strings.HasPrefix(file, "backend/cmd/test-server/") ||
		strings.HasPrefix(file, "backend/internal/testutil/")
}

// isTSTestPath checks if a file path should be categorized as TypeScript test code
func isTSTestPath(file string) bool {
	return strings.HasPrefix(file, "e2e/") ||
		strings.HasPrefix(file, "frontend/src/test/")
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
