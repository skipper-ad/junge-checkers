package gen

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	junge "github.com/skipper-ad/junge-checkers"
)

func RandInt(min, max int) int {
	if min > max {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		panic(fmt.Sprintf("secure random failed: %v", err))
	}
	return min + int(n.Int64())
}

func Sample[T any](values []T) T {
	if len(values) == 0 {
		panic("gen.Sample called with empty slice")
	}
	return values[RandInt(0, len(values)-1)]
}

func Bytes(c *junge.C, length int) []byte {
	data := make([]byte, length)
	if _, err := rand.Read(data); err != nil {
		c.CheckFailed("Checker failed", fmt.Sprintf("generate random bytes: %v", err))
	}
	return data
}

func String(length int) string {
	return StringAlphabet(length, AlphaNumericAlphabet)
}

func StringAlphabet(length int, alphabet string) string {
	if length <= 0 {
		return ""
	}
	if alphabet == "" {
		panic("empty alphabet")
	}
	var b strings.Builder
	b.Grow(length)
	for i := 0; i < length; i++ {
		b.WriteByte(alphabet[RandInt(0, len(alphabet)-1)])
	}
	return b.String()
}

func Username(saltLength ...int) string {
	salt := 5
	if len(saltLength) > 0 {
		salt = saltLength[0]
	}
	return Sample(Usernames) + StringAlphabet(salt, AlphaLowerAlphabet)
}

func Password(length int) string {
	return String(length)
}

func UserAgent() string {
	return Sample(UserAgents)
}

func Word() string {
	return StringAlphabet(RandInt(3, 10), AlphaLowerAlphabet)
}

func Words(count int) string {
	words := make([]string, 0, count)
	for i := 0; i < count; i++ {
		words = append(words, Word())
	}
	return strings.Join(words, " ")
}

func Sentence() string {
	size := RandInt(5, 15)
	words := make([]string, 0, size)
	for i := 0; i < size; i++ {
		word := Word()
		if i == 0 {
			word = capitalize(word)
		}
		if i == size-1 {
			word += "."
		}
		words = append(words, word)
	}
	return strings.Join(words, " ")
}

func Sentences(count int) string {
	items := make([]string, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, Sentence())
	}
	return strings.Join(items, " ")
}

func Paragraph() string {
	return Sentences(RandInt(2, 6))
}

func capitalize(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
