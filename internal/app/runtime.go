package app

func getGithubTokenWithRuntime(t string, runtimePort RuntimePort) string {
	if t == "" {
		return runtimePort.Getenv("GITHUB_TOKEN")
	}

	return t
}
