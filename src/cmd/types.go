package cmd

// GlobalFlags holds CLI flags shared across commands.
type GlobalFlags struct {
	Model        string
	SystemPrompt string
	MaxTokens    int
	LogLevel     string
	APIKey       string
	BotToken     string
	StorePath    string
	OutputMethod string
}

var flags GlobalFlags
