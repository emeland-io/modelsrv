package capacity

import (
	"fmt"
	"math/big"
	"strings"
)

// Amount is a non-negative decimal quantity stored as a canonical string.
type Amount string

// ParseAmount validates and normalizes a non-negative decimal amount.
func ParseAmount(s string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("amount is required")
	}
	rat, ok := new(big.Rat).SetString(s)
	if !ok {
		return "", fmt.Errorf("invalid amount %q", s)
	}
	if rat.Sign() < 0 {
		return "", fmt.Errorf("amount must be non-negative")
	}
	return Amount(s), nil
}
