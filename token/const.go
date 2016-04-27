package token

// Default settings for all generators
const (
	DefaultLength             = 6   // Default length of the generated tokens
	DefaultPeriod             = 30  // Default time period for TOTP tokens, in seconds
	DefaultRandomSecretLength = 100 // Default random secret length
	DefaultWindowBack         = 1   // Default TOTP verification window back steps
	DefaultWindowForward      = 1   // Default TOTP verification window forward steps
)

// Maximum values for all generators
const (
	MaxLength = 10 // Maximum token length
)
