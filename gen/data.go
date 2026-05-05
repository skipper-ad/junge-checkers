package gen

import (
	_ "embed"
	"strings"
)

//go:embed data/usernames.txt
var usernamesRaw string

//go:embed data/useragents.txt
var userAgentsRaw string

const (
	AlphaAlphabet        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphaLowerAlphabet   = "abcdefghijklmnopqrstuvwxyz"
	AlphaUpperAlphabet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	AlphaNumericAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	HexAlphabet          = "0123456789abcdef"
	PrintableAlphabet    = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ "
)

var (
	Usernames  = splitLines(usernamesRaw)
	UserAgents = splitLines(userAgentsRaw)
)

func splitLines(raw string) []string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
