package stats

import (
	"fmt"
	"sort"
	"time"

	"github.com/spideyz0r/fh/pkg/storage"
)

// Stats contains aggregated statistics about command history
type Stats struct {
	TotalCommands    int64
	UniqueCommands   int64
	SuccessRate      float64
	AvgPerDay        float64
	TopCommands      []CommandCount
	CommandsByDir    []DirectoryCount
	TimeDistribution map[int]int // hour -> count
	FirstCommand     time.Time
	LastCommand      time.Time
}

// CommandCount represents a command and how many times it was executed
type CommandCount struct {
	Command string
	Count   int
}

// DirectoryCount represents a directory and command count
type DirectoryCount struct {
	Directory string
	Count     int
}

// Collect gathers statistics from the database
func Collect(db storage.Store) (*Stats, error) {
	stats := &Stats{
		TimeDistribution: make(map[int]int),
	}

	// Get all entries
	entries, err := db.Query(storage.QueryFilters{Limit: 0}) // 0 = unlimited
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}

	if len(entries) == 0 {
		return stats, nil
	}

	// Calculate basic stats
	stats.TotalCommands = int64(len(entries))

	// Track unique commands, directories, success count
	uniqueCommands := make(map[string]int)
	directories := make(map[string]int)
	successCount := 0

	var firstTimestamp, lastTimestamp int64
	firstTimestamp = entries[0].Timestamp
	lastTimestamp = entries[0].Timestamp

	for _, entry := range entries {
		// Unique commands
		uniqueCommands[entry.Command]++

		// Directories
		if entry.Cwd != "" {
			directories[entry.Cwd]++
		}

		// Success rate
		if entry.ExitCode == 0 {
			successCount++
		}

		// Time distribution (hour of day)
		t := time.Unix(entry.Timestamp, 0)
		hour := t.Hour()
		stats.TimeDistribution[hour]++

		// Track first/last timestamps
		if entry.Timestamp < firstTimestamp {
			firstTimestamp = entry.Timestamp
		}
		if entry.Timestamp > lastTimestamp {
			lastTimestamp = entry.Timestamp
		}
	}

	stats.UniqueCommands = int64(len(uniqueCommands))
	stats.SuccessRate = float64(successCount) / float64(stats.TotalCommands) * 100

	// Calculate average per day
	stats.FirstCommand = time.Unix(firstTimestamp, 0)
	stats.LastCommand = time.Unix(lastTimestamp, 0)
	daysDiff := stats.LastCommand.Sub(stats.FirstCommand).Hours() / 24
	if daysDiff > 0 {
		stats.AvgPerDay = float64(stats.TotalCommands) / daysDiff
	} else {
		stats.AvgPerDay = float64(stats.TotalCommands)
	}

	// Build top commands list
	stats.TopCommands = make([]CommandCount, 0, len(uniqueCommands))
	for cmd, count := range uniqueCommands {
		stats.TopCommands = append(stats.TopCommands, CommandCount{
			Command: cmd,
			Count:   count,
		})
	}

	// Sort by count (descending)
	sort.Slice(stats.TopCommands, func(i, j int) bool {
		return stats.TopCommands[i].Count > stats.TopCommands[j].Count
	})

	// Build directories list
	stats.CommandsByDir = make([]DirectoryCount, 0, len(directories))
	for dir, count := range directories {
		stats.CommandsByDir = append(stats.CommandsByDir, DirectoryCount{
			Directory: dir,
			Count:     count,
		})
	}

	// Sort by count (descending)
	sort.Slice(stats.CommandsByDir, func(i, j int) bool {
		return stats.CommandsByDir[i].Count > stats.CommandsByDir[j].Count
	})

	return stats, nil
}

// CollectFiltered gathers statistics with filters applied
func CollectFiltered(db storage.Store, filters storage.QueryFilters) (*Stats, error) {
	// For filtered stats, we need a custom implementation
	// that queries with filters first, then calculates stats

	// Get filtered entries
	entries, err := db.Query(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries: %w", err)
	}

	stats := &Stats{
		TimeDistribution: make(map[int]int),
	}

	if len(entries) == 0 {
		return stats, nil
	}

	// Calculate stats on filtered entries
	stats.TotalCommands = int64(len(entries))

	uniqueCommands := make(map[string]int)
	directories := make(map[string]int)
	successCount := 0

	var firstTimestamp, lastTimestamp int64
	firstTimestamp = entries[0].Timestamp
	lastTimestamp = entries[0].Timestamp

	for _, entry := range entries {
		uniqueCommands[entry.Command]++

		if entry.Cwd != "" {
			directories[entry.Cwd]++
		}

		if entry.ExitCode == 0 {
			successCount++
		}

		t := time.Unix(entry.Timestamp, 0)
		hour := t.Hour()
		stats.TimeDistribution[hour]++

		if entry.Timestamp < firstTimestamp {
			firstTimestamp = entry.Timestamp
		}
		if entry.Timestamp > lastTimestamp {
			lastTimestamp = entry.Timestamp
		}
	}

	stats.UniqueCommands = int64(len(uniqueCommands))
	if stats.TotalCommands > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalCommands) * 100
	}

	stats.FirstCommand = time.Unix(firstTimestamp, 0)
	stats.LastCommand = time.Unix(lastTimestamp, 0)
	daysDiff := stats.LastCommand.Sub(stats.FirstCommand).Hours() / 24
	if daysDiff > 0 {
		stats.AvgPerDay = float64(stats.TotalCommands) / daysDiff
	} else {
		stats.AvgPerDay = float64(stats.TotalCommands)
	}

	// Build top commands list
	stats.TopCommands = make([]CommandCount, 0, len(uniqueCommands))
	for cmd, count := range uniqueCommands {
		stats.TopCommands = append(stats.TopCommands, CommandCount{
			Command: cmd,
			Count:   count,
		})
	}
	sort.Slice(stats.TopCommands, func(i, j int) bool {
		return stats.TopCommands[i].Count > stats.TopCommands[j].Count
	})

	// Build directories list
	stats.CommandsByDir = make([]DirectoryCount, 0, len(directories))
	for dir, count := range directories {
		stats.CommandsByDir = append(stats.CommandsByDir, DirectoryCount{
			Directory: dir,
			Count:     count,
		})
	}
	sort.Slice(stats.CommandsByDir, func(i, j int) bool {
		return stats.CommandsByDir[i].Count > stats.CommandsByDir[j].Count
	})

	return stats, nil
}

// Format formats statistics for display
func (s *Stats) Format(topN int) string {
	if s.TotalCommands == 0 {
		return "No commands in history yet."
	}

	result := fmt.Sprintf("fh - History Statistics\n")
	result += fmt.Sprintf("=======================\n\n")

	result += fmt.Sprintf("Total Commands:   %d\n", s.TotalCommands)
	result += fmt.Sprintf("Unique Commands:  %d\n", s.UniqueCommands)
	result += fmt.Sprintf("Success Rate:     %.1f%%\n", s.SuccessRate)
	result += fmt.Sprintf("Avg Per Day:      %.1f\n", s.AvgPerDay)
	result += fmt.Sprintf("First Command:    %s\n", s.FirstCommand.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("Last Command:     %s\n\n", s.LastCommand.Format("2006-01-02 15:04:05"))

	// Top N commands
	if len(s.TopCommands) > 0 {
		result += fmt.Sprintf("Top %d Commands:\n", min(topN, len(s.TopCommands)))
		result += fmt.Sprintf("----------------\n")
		for i := 0; i < min(topN, len(s.TopCommands)); i++ {
			cmd := s.TopCommands[i]
			percentage := float64(cmd.Count) / float64(s.TotalCommands) * 100
			// Truncate long commands
			displayCmd := cmd.Command
			if len(displayCmd) > 60 {
				displayCmd = displayCmd[:57] + "..."
			}
			result += fmt.Sprintf("%3d. (%3d | %5.1f%%) %s\n", i+1, cmd.Count, percentage, displayCmd)
		}
		result += "\n"
	}

	// Top directories
	if len(s.CommandsByDir) > 0 {
		result += fmt.Sprintf("Top %d Directories:\n", min(5, len(s.CommandsByDir)))
		result += fmt.Sprintf("-------------------\n")
		for i := 0; i < min(5, len(s.CommandsByDir)); i++ {
			dir := s.CommandsByDir[i]
			percentage := float64(dir.Count) / float64(s.TotalCommands) * 100
			result += fmt.Sprintf("%3d. (%3d | %5.1f%%) %s\n", i+1, dir.Count, percentage, dir.Directory)
		}
		result += "\n"
	}

	// Hour distribution
	if len(s.TimeDistribution) > 0 {
		result += "Commands by Hour:\n"
		result += "-----------------\n"
		result += formatHourDistribution(s.TimeDistribution, s.TotalCommands)
	}

	return result
}

// formatHourDistribution creates a visual histogram of command distribution by hour
func formatHourDistribution(dist map[int]int, total int64) string {
	result := ""

	// Find max count for scaling
	maxCount := 0
	for _, count := range dist {
		if count > maxCount {
			maxCount = count
		}
	}

	// Display each hour
	for hour := 0; hour < 24; hour++ {
		count := dist[hour]
		if count == 0 {
			continue
		}

		// Scale to 40 characters max
		barLength := 0
		if maxCount > 0 {
			barLength = (count * 40) / maxCount
		}

		bar := ""
		for i := 0; i < barLength; i++ {
			bar += "█"
		}

		percentage := float64(count) / float64(total) * 100
		result += fmt.Sprintf("%02d:00 (%3d | %5.1f%%) %s\n", hour, count, percentage, bar)
	}

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
