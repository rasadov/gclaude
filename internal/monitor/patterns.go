package monitor

import "regexp"

var defaultPatterns = []string{
	`\[Y/n\]`,
	`\[y/N\]`,
	`\(y/n\)`,
	`\(Y/n\)`,
	`\?\s*$`,
	`Press Enter`,
	`press enter`,
	`Choose.*:\s*$`,
	`Select.*:\s*$`,
	`Enter.*:\s*$`,
	`Type.*:\s*$`,
	`waiting for.*input`,
	`Waiting for.*input`,
	`continue\?`,
	`proceed\?`,
	`confirm`,
	`>>\s*$`,
	`❯\s*$`,
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
