package cmd

// globalFlags holds CLI flags shared across commands.
type globalFlags struct {
	model        string
	systemPrompt string
	maxTokens    int
	logLevel     string
	apiKey       string
	botToken     string
}

var flags globalFlags
