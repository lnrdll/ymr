package app

func getGithubToken(t string) string {
	return getGithubTokenWithRuntime(t, newDefaultDeps().runtime)
}

func getGithubTokenWithRuntime(t string, runtimePort RuntimePort) string {
	if t == "" {
		return runtimePort.Getenv("GITHUB_TOKEN")
	}

	return t
}
