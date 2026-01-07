package preflight

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/robertguss/bmad-automate-go/internal/config"
)

// CheckResult represents the result of a single pre-flight check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
	Error   string
}

// Results holds all pre-flight check results
type Results struct {
	Checks  []CheckResult
	AllPass bool
}

// RunAll executes all pre-flight checks
func RunAll(cfg *config.Config) *Results {
	results := &Results{
		Checks:  make([]CheckResult, 0),
		AllPass: true,
	}

	// Check Claude CLI
	results.addCheck(checkClaudeCLI())

	// Check sprint-status.yaml exists
	results.addCheck(checkSprintStatus(cfg))

	// Check story directory exists
	results.addCheck(checkStoryDir(cfg))

	// Check working directory is a git repo
	results.addCheck(checkGitRepo(cfg))

	// Check for uncommitted changes (warning only)
	gitCheck := checkGitClean(cfg)
	results.addCheck(gitCheck)

	return results
}

// addCheck adds a check result and updates AllPass
func (r *Results) addCheck(check CheckResult) {
	r.Checks = append(r.Checks, check)
	if !check.Passed && check.Name != "Git Clean" {
		// Git clean is a warning, not a blocker
		r.AllPass = false
	}
}

// PassedCount returns the number of passed checks
func (r *Results) PassedCount() int {
	count := 0
	for _, check := range r.Checks {
		if check.Passed {
			count++
		}
	}
	return count
}

// FailedChecks returns only the failed checks
func (r *Results) FailedChecks() []CheckResult {
	failed := make([]CheckResult, 0)
	for _, check := range r.Checks {
		if !check.Passed {
			failed = append(failed, check)
		}
	}
	return failed
}

// checkClaudeCLI verifies the Claude CLI is installed and accessible
func checkClaudeCLI() CheckResult {
	result := CheckResult{Name: "Claude CLI"}

	cmd := exec.Command("which", "claude")
	output, err := cmd.Output()

	if err != nil {
		result.Passed = false
		result.Error = "Claude CLI not found in PATH"
		return result
	}

	path := strings.TrimSpace(string(output))
	result.Passed = true
	result.Message = fmt.Sprintf("Found at %s", path)

	// Try to get version
	versionCmd := exec.Command("claude", "--version")
	versionOutput, err := versionCmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(versionOutput))
		result.Message = fmt.Sprintf("v%s", version)
	}

	return result
}

// checkSprintStatus verifies the sprint-status.yaml file exists
func checkSprintStatus(cfg *config.Config) CheckResult {
	result := CheckResult{Name: "Sprint Status"}

	if _, err := os.Stat(cfg.SprintStatusPath); os.IsNotExist(err) {
		result.Passed = false
		result.Error = fmt.Sprintf("File not found: %s", cfg.SprintStatusPath)
		return result
	}

	result.Passed = true
	result.Message = "Found"
	return result
}

// checkStoryDir verifies the story directory exists
func checkStoryDir(cfg *config.Config) CheckResult {
	result := CheckResult{Name: "Story Directory"}

	info, err := os.Stat(cfg.StoryDir)
	if os.IsNotExist(err) {
		result.Passed = false
		result.Error = fmt.Sprintf("Directory not found: %s", cfg.StoryDir)
		return result
	}

	if !info.IsDir() {
		result.Passed = false
		result.Error = fmt.Sprintf("Not a directory: %s", cfg.StoryDir)
		return result
	}

	result.Passed = true
	result.Message = "Found"
	return result
}

// checkGitRepo verifies the working directory is a git repository
func checkGitRepo(cfg *config.Config) CheckResult {
	result := CheckResult{Name: "Git Repository"}

	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = cfg.WorkingDir

	if err := cmd.Run(); err != nil {
		result.Passed = false
		result.Error = "Not a git repository"
		return result
	}

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = cfg.WorkingDir
	branchOutput, err := branchCmd.Output()

	if err == nil {
		branch := strings.TrimSpace(string(branchOutput))
		result.Message = fmt.Sprintf("Branch: %s", branch)
	} else {
		result.Message = "Found"
	}

	result.Passed = true
	return result
}

// checkGitClean checks for uncommitted changes (warning only)
func checkGitClean(cfg *config.Config) CheckResult {
	result := CheckResult{Name: "Git Clean"}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cfg.WorkingDir
	output, err := cmd.Output()

	if err != nil {
		result.Passed = true
		result.Message = "Unable to check"
		return result
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		result.Passed = false
		result.Error = "Uncommitted changes detected"
		return result
	}

	result.Passed = true
	result.Message = "Working tree clean"
	return result
}

// GetGitBranch returns the current git branch name
func GetGitBranch(workingDir string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// IsGitClean returns true if the git working tree is clean
func IsGitClean(workingDir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) == 0
}
