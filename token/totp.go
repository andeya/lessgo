package token

import "time"

// TOTP is used to generate tokens based on RFC-6238.
//
// Example:
//
//  totp := &TOTP{Secret: "your-secret", IsBase32Secret: true}
//  token := totp.Get()
//
// TOTP assumes a set of default values for Secret, Length, Time, Period, WindowBack, WindowForward and IsBase32Secret
//
// If no Secret is informed, TOTP will generate a random one that you need to store with the Counter, for future token
// verifications.
//
// Check this package constants to see the current default values.
type TOTP struct {
	Secret         string    // The secret used to generate a token
	Length         uint8     // The token length
	Time           time.Time // The time used to generate the token
	IsBase32Secret bool      //
	Period         uint8     // The step size to slice time, in seconds
	WindowBack     uint8     // How many steps HOTP will go backwards to validate a token
	WindowForward  uint8     // How many steps HOTP will go forward to validate a token
}

func (t *TOTP) setDefaults() {
	if len(t.Secret) == 0 {
		t.Secret = generateRandomSecret(DefaultRandomSecretLength, t.IsBase32Secret)
	}
	if t.Length == 0 {
		t.Length = DefaultLength
	}
	if t.Time.IsZero() {
		t.Time = time.Now()
	}
	if t.Period == 0 {
		t.Period = DefaultPeriod
	}
	if t.WindowBack == 0 {
		t.WindowBack = DefaultWindowBack
	}
	if t.WindowForward == 0 {
		t.WindowForward = DefaultWindowForward
	}
}

func (t *TOTP) normalize() {
	if t.Length > MaxLength {
		t.Length = MaxLength
	}
}

// Get a time-based token
func (t *TOTP) Get() string {
	t.setDefaults()
	t.normalize()
	ts := uint64(t.Time.Unix() / int64(t.Period))
	hotp := &HOTP{Secret: t.Secret, Counter: ts, Length: t.Length, IsBase32Secret: t.IsBase32Secret}
	return hotp.Get()
}

// Now is a fluent interface to set the TOTP generator's time to the current date/time
func (t *TOTP) Now() *TOTP {
	t.Time = time.Now()
	return t
}

// Verify a token with the current settings, including the WindowBack and WindowForward
func (t TOTP) Verify(token string) bool {
	t.setDefaults()
	t.normalize()
	givenTime := t.Time
	for i := int(t.WindowBack) * -1; i <= int(t.WindowForward); i++ {
		t.Time = givenTime.Add(time.Second * time.Duration(int(t.Period)*i))
		if t.Get() == token {
			return true
		}
	}
	return false
}
