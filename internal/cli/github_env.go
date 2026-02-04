package cli

import "github.com/boinger/confvis/internal/checks"

// ParseRepoFromEnv extracts owner and repo from GITHUB_REPOSITORY environment.
// Returns empty strings if already have both owner and repo, or if env is empty.
func ParseRepoFromEnv(currentOwner, currentRepo string, env *checks.GitHubEnv) (owner, repo string) {
	// Skip if we already have both values
	if currentOwner != "" && currentRepo != "" {
		return currentOwner, currentRepo
	}

	// Skip if no environment repository
	if env == nil || env.Repository == "" {
		return currentOwner, currentRepo
	}

	// Parse repository
	parsedOwner, parsedRepo, err := checks.ParseRepository(env.Repository)
	if err != nil {
		return currentOwner, currentRepo
	}

	// Fill in missing values
	if currentOwner == "" {
		currentOwner = parsedOwner
	}
	if currentRepo == "" {
		currentRepo = parsedRepo
	}

	return currentOwner, currentRepo
}
