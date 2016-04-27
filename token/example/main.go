package main

import (
	"encoding/base32"
	"flag"
	"fmt"
	"strings"

	"github.com/lessgo/lessgo/token"
)

var (
	secret   = flag.String("secret", "AAAABBBBCCCCDDDD", "Secret key")
	isBase32 = flag.Bool("base32", true, "If true, the secret is interpreted as a Base32 string")
	length   = flag.Uint("length", token.DefaultLength, "OTP length")
	period   = flag.Uint("period", token.DefaultPeriod, "Period in seconds")
	counter  = flag.Uint64("counter", 0, "Counter")
)

func main() {
	flag.Parse()

	key := *secret
	if !*isBase32 {
		key = base32.StdEncoding.EncodeToString([]byte(*secret))
	}

	key = strings.ToUpper(key)
	if !isGoogleAuthenticatorCompatible(key) {
		fmt.Println("WARN: Google Authenticator requires 16 chars base32 secret, without padding")
	}

	fmt.Println("Secret Base32 Encoded Key: ", key)

	totp := &token.TOTP{
		Secret:         key,
		Length:         uint8(*length),
		Period:         uint8(*period),
		IsBase32Secret: true,
	}
	fmt.Println("TOTP:", totp.Get(), totp.Verify(totp.Get()))

	hotp := &token.HOTP{
		Secret:         key,
		Length:         uint8(*length),
		Counter:        *counter,
		IsBase32Secret: true,
	}

	fmt.Println("HOTP:", hotp.Get())
}

func isGoogleAuthenticatorCompatible(base32Secret string) bool {
	cleaned := strings.Replace(base32Secret, "=", "", -1)
	cleaned = strings.Replace(cleaned, " ", "", -1)
	return len(cleaned) == 16
}
