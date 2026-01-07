package git

import (
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Status represents the current git repository status
type Status struct {
	Branch           string
	IsClean          bool
	HasUncommitted   bool
	HasUntracked     bool
	Ahead            int
	Behind           int
	LastFetch        time.Time
	IsGitRepo        bool
	UncommittedCount int
	UntrackedCount   int
}

// StatusMsg is sent when git status is updated
type StatusMsg struct {
	Status Status
	Error  error
}

// GetStatus retrieves the current git status
func GetStatus(workDir string) Status {
	status := Status{
		IsGitRepo: isGitRepo(workDir),
	}

	if !status.IsGitRepo {
		return status
	}

	status.Branch = getBranch(workDir)
	status.HasUncommitted, status.UncommittedCount = hasUncommitted(workDir)
	status.HasUntracked, status.UntrackedCount = hasUntracked(workDir)
	status.IsClean = !status.HasUncommitted && !status.HasUntracked
	status.Ahead, status.Behind = getAheadBehind(workDir)

	return status
}

// GetStatusCmd returns a command that fetches git status
func GetStatusCmd(workDir string) tea.Cmd {
	return func() tea.Msg {
		status := GetStatus(workDir)
		return StatusMsg{Status: status}
	}
}

// isGitRepo checks if the directory is a git repository
func isGitRepo(workDir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// getBranch gets the current branch name
func getBranch(workDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// hasUncommitted checks for uncommitted changes
func hasUncommitted(workDir string) (bool, int) {
	cmd := exec.Command("git", "diff", "--shortstat")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return false, 0
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		// Also check staged changes
		cmd = exec.Command("git", "diff", "--cached", "--shortstat")
		cmd.Dir = workDir
		output, err = cmd.Output()
		if err != nil {
			return false, 0
		}
		text = strings.TrimSpace(string(output))
	}

	if text == "" {
		return false, 0
	}

	// Count changed files
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = workDir
	output, _ = cmd.Output()
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	count := 0
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, "??") {
			count++
		}
	}

	return true, count
}

// hasUntracked checks for untracked files
func hasUntracked(workDir string) (bool, int) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return false, 0
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		return false, 0
	}

	lines := strings.Split(text, "\n")
	return true, len(lines)
}

// getAheadBehind gets the number of commits ahead/behind remote
func getAheadBehind(workDir string) (ahead, behind int) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0
	}

	// Parse behind and ahead
	// git rev-list --left-right gives <behind>\t<ahead>
	fmt := "%d"
	_, _ = strings.NewReader(parts[0]).Read([]byte(fmt))
	var b, a int
	_, _ = strings.NewReader(parts[0]).Read([]byte(fmt))
	if n, _ := strings.NewReader(parts[0]).Read([]byte(fmt)); n > 0 {
		b = parseIntOrZero(parts[0])
	}
	a = parseIntOrZero(parts[1])

	return a, b
}

func parseIntOrZero(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// FormatStatus returns a human-readable status string
func (s Status) FormatStatus() string {
	if !s.IsGitRepo {
		return "Not a git repo"
	}

	if s.IsClean {
		return "Clean"
	}

	var parts []string
	if s.HasUncommitted {
		parts = append(parts, "Modified")
	}
	if s.HasUntracked {
		parts = append(parts, "Untracked")
	}

	return strings.Join(parts, ", ")
}

// FormatBranch returns the branch with ahead/behind info
func (s Status) FormatBranch() string {
	if !s.IsGitRepo {
		return ""
	}

	branch := s.Branch
	if s.Ahead > 0 || s.Behind > 0 {
		if s.Ahead > 0 && s.Behind > 0 {
			branch += " ↑" + string(rune('0'+s.Ahead)) + "↓" + string(rune('0'+s.Behind))
		} else if s.Ahead > 0 {
			branch += " ↑" + string(rune('0'+s.Ahead))
		} else {
			branch += " ↓" + string(rune('0'+s.Behind))
		}
	}

	return branch
}
