package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// CommitData represents a single commit's data
type CommitData struct {
	Hash       string
	Date       time.Time
	Message    string
	TotalTasks int
	DoneTasks  int
}

// DailyData represents aggregated data for a single day
type DailyData struct {
	Date       time.Time
	TotalTasks int
	DoneTasks  int
	Message    string
}

func main() {
	// Validate we're in a git repository
	if err := validateGitRepo(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get all commits where ROADMAP.md changed
	commits, err := getCommitsForRoadmap()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error getting commits: %v\n", err)
		os.Exit(1)
	}

	if len(commits) == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "No commits found where ROADMAP.md changed\n")
		os.Exit(1)
	}

	// Process each commit to count tasks
	for i := range commits {
		content, err := getRoadmapAtCommit(commits[i].Hash)
		if err != nil {
			// Skip commits where the file doesn't exist
			continue
		}
		total, done := countTasks(content)
		commits[i].TotalTasks = total
		commits[i].DoneTasks = done
	}

	// Aggregate by day (use the latest commit per day)
	dailyData := aggregateByDay(commits)

	// Fill gaps (days with no commits use previous day's data)
	filledData := fillGaps(dailyData)

	// Output CSV
	outputCSV(filledData)
}

// validateGitRepo checks if we're in a git repository
func validateGitRepo() error {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not in a git repository: %v", err)
	}
	return nil
}

// getCommitsForRoadmap gets all commits where ROADMAP.md changed
// Returns commits sorted by date (oldest first)
func getCommitsForRoadmap() ([]CommitData, error) {
	// Use --follow to track file renames
	cmd := exec.Command("git", "log", "--follow", "--format=%H|%ai|%s", "--", "ROADMAP.md")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git log: %v", err)
	}

	var commits []CommitData
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

		hash := parts[0]
		dateStr := parts[1]
		message := parts[2]

		// Parse date (format: 2025-01-15 10:30:00 +0100)
		date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			// Try alternative format without the timezone
			date, err = time.Parse("2006-01-02 15:04:05", dateStr)
			if err != nil {
				continue
			}
		}

		commits = append(commits, CommitData{
			Hash:    hash,
			Date:    date,
			Message: message,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading git log output: %v", err)
	}

	// Sort by date (oldest first)
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date.Before(commits[j].Date)
	})

	return commits, nil
}

// getRoadmapAtCommit gets the content of ROADMAP.md at a specific commit
func getRoadmapAtCommit(hash string) (string, error) {
	cmd := exec.Command("git", "show", hash+":ROADMAP.md")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %v", err)
	}
	return string(output), nil
}

// countTasks counts total and done tasks in markdown content
// Supports both * [ ] and - [ ] formats
func countTasks(content string) (total, done int) {
	// Match both * [x], * [ ], - [x], - [ ] patterns
	// Pattern: optional whitespace, then * or -, then space, then [x] or [ ]
	pattern := regexp.MustCompile(`(?m)^\s*[-*]\s+\[([ x])]`)
	matches := pattern.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		total++
		if match[1] == "x" {
			done++
		}
	}

	return total, done
}

// aggregateByDay groups commits by date and uses the latest commit per day
func aggregateByDay(commits []CommitData) []DailyData {
	// Map: date (YYYY-MM-DD) -> latest commit data for that day
	dailyMap := make(map[string]CommitData)

	for _, commit := range commits {
		dateKey := commit.Date.Format("2006-01-02")
		existing, exists := dailyMap[dateKey]

		// Use the latest commit if multiple commits on the same day
		if !exists || commit.Date.After(existing.Date) {
			dailyMap[dateKey] = commit
		}
	}

	// Convert map to slice and sort by date
	var dailyData []DailyData
	for _, commit := range dailyMap {
		dailyData = append(dailyData, DailyData{
			Date:       commit.Date,
			TotalTasks: commit.TotalTasks,
			DoneTasks:  commit.DoneTasks,
			Message:    commit.Message,
		})
	}

	sort.Slice(dailyData, func(i, j int) bool {
		return dailyData[i].Date.Before(dailyData[j].Date)
	})

	return dailyData
}

// fillGaps fills days with no commits using the previous day's data
func fillGaps(dailyData []DailyData) []DailyData {
	if len(dailyData) == 0 {
		return dailyData
	}

	// Find date range - normalize to start of day
	startDate := time.Date(
		dailyData[0].Date.Year(),
		dailyData[0].Date.Month(),
		dailyData[0].Date.Day(),
		0, 0, 0, 0,
		dailyData[0].Date.Location(),
	)

	// The end date is today, normalized to start of day
	now := time.Now()
	endDate := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0, 0, 0, 0,
		now.Location(),
	)

	// Create a map for quick lookups
	dataMap := make(map[string]DailyData)
	for _, data := range dailyData {
		dateKey := data.Date.Format("2006-01-02")
		dataMap[dateKey] = data
	}

	// Fill gaps
	var filled []DailyData
	var lastData DailyData
	hasLastData := false

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateKey := d.Format("2006-01-02")
		if data, exists := dataMap[dateKey]; exists {
			lastData = data
			hasLastData = true
			filled = append(filled, data)
		} else if hasLastData {
			// Use previous day's data but update the date
			filled = append(filled, DailyData{
				Date:       d,
				TotalTasks: lastData.TotalTasks,
				DoneTasks:  lastData.DoneTasks,
				Message:    lastData.Message,
			})
		}
	}

	return filled
}

// outputCSV writes the data as CSV to stdout
func outputCSV(data []DailyData) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"date", "total_tasks", "done_tasks", "commit_message"}
	if err := writer.Write(header); err != nil {
		_, err := fmt.Fprintf(os.Stderr, "Error writing CSV header: %v\n", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	// Write data rows
	for _, d := range data {
		row := []string{
			d.Date.Format("2006-01-02"),
			fmt.Sprintf("%d", d.TotalTasks),
			fmt.Sprintf("%d", d.DoneTasks),
			d.Message,
		}
		if err := writer.Write(row); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error writing CSV row: %v\n", err)
			os.Exit(1)
		}
	}
}
