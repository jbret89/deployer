package httpapi

import "regexp"

var repoNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidRepoName(repo string) bool {
	return repoNamePattern.MatchString(repo)
}
