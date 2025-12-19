package monitor

import "regexp"

var defaultPatterns = []string{
	// Yes/No prompts
	`\[Y/n\]`,
	`\[y/N\]`,
	`\(y/n\)`,
	`\(Y/n\)`,
	// Questions
	`\?\s*$`,
	`Do you want to`,
	// Action prompts
	`Press Enter`,
	`press enter`,
	`Choose.*:\s*$`,
	`Select.*:\s*$`,
	`Enter.*:\s*$`,
	`Type.*:\s*$`,
	// Waiting states
	`waiting for.*input`,
	`Waiting for.*input`,
	// Confirmations
	`continue\?`,
	`proceed\?`,
	`confirm`,
	// CLI selectors (Claude Code uses these)
	`❯\s+\d+\.`,         // ❯ 1. Yes
	`^\s*❯`,             // Line starting with ❯
	`>>\s*$`,
	// Claude Code specific
	`Create file`,
	`Edit file`,
	`Run command`,
	`Allow once`,
	`Allow all`,
}

var compiledPatterns []*regexp.Regexp

func init() {
	compiledPatterns = make([]*regexp.Regexp, 0, len(defaultPatterns))
	for _, p := range defaultPatterns {
		if re, err := regexp.Compile(p); err == nil {
			compiledPatterns = append(compiledPatterns, re)
		}
	}
}

func MatchesInputPattern(output string) bool {
	for _, re := range compiledPatterns {
		if re.MatchString(output) {
			return true
		}
	}
	return false
}

func GetMatchedPattern(output string) string {
	for _, re := range compiledPatterns {
		if match := re.FindString(output); match != "" {
			return match
		}
	}
	return ""
}

func AddPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	compiledPatterns = append(compiledPatterns, re)
	return nil
}
