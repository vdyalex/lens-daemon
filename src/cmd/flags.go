package cmd

import "strconv"

// flagEnvPair holds a single environment variable name and its string value.
type flagEnvPair struct {
	key   string
	value string
}

// flagEnvPairs returns the ordered set of flag-to-env mappings derived from flags.
// Only flags with non-zero values are included.
// Callers consume the pairs according to their context (os.Setenv vs subprocess env).
func flagEnvPairs() []flagEnvPair {
	var pairs []flagEnvPair

	if flags.model != "" {
		pairs = append(pairs, flagEnvPair{"ANTHROPIC_MODEL", flags.model})
	}
	if flags.systemPrompt != "" {
		pairs = append(pairs, flagEnvPair{"ANTHROPIC_SYSTEM_PROMPT", flags.systemPrompt})
	}
	if flags.maxTokens != 0 {
		pairs = append(pairs, flagEnvPair{"ANTHROPIC_MAX_RESPONSE_TOKENS", strconv.Itoa(flags.maxTokens)})
	}
	if flags.logLevel != "" {
		pairs = append(pairs, flagEnvPair{"LOG_LEVEL", flags.logLevel})
	}
	if flags.apiKey != "" {
		pairs = append(pairs, flagEnvPair{"ANTHROPIC_API_KEY", flags.apiKey})
	}
	if flags.botToken != "" {
		pairs = append(pairs, flagEnvPair{"TELEGRAM_BOT_TOKEN", flags.botToken})
	}

	return pairs
}
