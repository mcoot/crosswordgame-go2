package model

// Bot strategy constants
const (
	BotStrategyRandom = "random"
)

// BotStrategyDisplayName returns a human-readable label for a strategy
func BotStrategyDisplayName(strategy string) string {
	switch strategy {
	case BotStrategyRandom:
		return "Random"
	default:
		return strategy
	}
}

// ValidBotStrategies returns all valid bot strategy names
func ValidBotStrategies() []string {
	return []string{BotStrategyRandom}
}
