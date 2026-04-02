package cmd

import "strconv"

// FlagEnvPair holds a single environment variable name and its string value.
type FlagEnvPair struct {
	Key   string
	Value string
}

// FlagEnvPairs returns the ordered set of flag-to-env mappings derived from flags.
// Only flags with non-zero values are included.
// Callers consume the pairs according to their context (os.Setenv vs subprocess env).
func FlagEnvPairs() []FlagEnvPair {
	var pairs []FlagEnvPair

	if flags.Model != "" {
		pairs = append(pairs, FlagEnvPair{"ANTHROPIC_MODEL", flags.Model})
	}
	if flags.SystemPrompt != "" {
		pairs = append(pairs, FlagEnvPair{"ANTHROPIC_SYSTEM_PROMPT", flags.SystemPrompt})
	}
	if flags.MaxTokens != 0 {
		pairs = append(pairs, FlagEnvPair{"ANTHROPIC_MAX_RESPONSE_TOKENS", strconv.Itoa(flags.MaxTokens)})
	}
	if flags.LogLevel != "" {
		pairs = append(pairs, FlagEnvPair{"LOG_LEVEL", flags.LogLevel})
	}
	if flags.APIKey != "" {
		pairs = append(pairs, FlagEnvPair{"ANTHROPIC_API_KEY", flags.APIKey})
	}
	if flags.BotToken != "" {
		pairs = append(pairs, FlagEnvPair{"TELEGRAM_BOT_TOKEN", flags.BotToken})
	}
	if flags.StorePath != "" {
		pairs = append(pairs, FlagEnvPair{"SUBSCRIBER_STORE_PATH", flags.StorePath})
	}

	return pairs
}
