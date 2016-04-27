package lessgo

import (
	"github.com/lessgo/lessgo/token"
)

// Token4226 is used to generate tokens based on RFC-4226.
func Token4226(secret string, length uint8, counter uint64, isBase32Secret bool) *token.HOTP {
	return &token.HOTP{
		Secret:         secret,         // The secret used to generate the token
		Length:         length,         // The token size, with a maximum determined by MaxLength
		Counter:        counter,        // The counter used as moving factor
		IsBase32Secret: isBase32Secret, // If true, the secret will be used as a Base32 encoded string
	}
}

// Token6238 is used to generate tokens based on RFC-6238.
func Token6238(secret string, length uint8, isBase32Secret bool, period uint8, windowBack uint8, windowForward uint8) *token.TOTP {
	return &token.TOTP{
		Secret:         secret,         // The secret used to generate a token
		Length:         length,         // The token length
		IsBase32Secret: isBase32Secret, // If true, the secret will be used as a Base32 encoded string
		Period:         period,         // The step size to slice time, in seconds
		WindowBack:     windowBack,     // How many steps HOTP will go backwards to validate a token
		WindowForward:  windowForward,  // How many steps HOTP will go forward to validate a token
	}
}

func SimpleToken6238(secret string, length uint8, isBase32Secret bool, period uint8) *token.TOTP {
	return &token.TOTP{
		Secret:         secret,         // The secret used to generate a token
		Length:         length,         // The token length
		IsBase32Secret: isBase32Secret, // If true, the secret will be used as a Base32 encoded string
		Period:         period,         // The step size to slice time, in seconds
	}
}
