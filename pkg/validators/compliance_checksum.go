//go:build !osspublic

package validators

import (
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

var digitsOnly = regexp.MustCompile(`\D`)

func digits(s string) string {
	return digitsOnly.ReplaceAllString(s, "")
}

// LuhnValid checks credit card numbers (Luhn algorithm).
func LuhnValid(s string) bool { return luhnValid(s) }

// luhnValid checks credit card numbers (Luhn algorithm).
func luhnValid(s string) bool {
	s = digits(s)
	if len(s) < 13 || len(s) > 19 {
		return false
	}
	sum, alt := 0, false
	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

// ibanValid validates IBAN mod-97 checksum.
func ibanValid(s string) bool {
	s = strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if len(s) < 15 || len(s) > 34 {
		return false
	}
	rearranged := s[4:] + s[:4]
	var expanded strings.Builder
	for _, r := range rearranged {
		if r >= 'A' && r <= 'Z' {
			expanded.WriteString(strconv.Itoa(int(r - 'A' + 10)))
		} else if r >= '0' && r <= '9' {
			expanded.WriteByte(byte(r))
		} else {
			return false
		}
	}
	rem := 0
	for _, r := range expanded.String() {
		if r < '0' || r > '9' {
			return false
		}
		rem = (rem*10 + int(r-'0')) % 97
	}
	return rem == 1
}

// peselValid validates Polish PESEL (11 digits + checksum).
// PeselValid checks Polish PESEL checksum (exported for proxy parity tests).
func PeselValid(s string) bool { return peselValid(s) }

func peselValid(s string) bool {
	s = digits(s)
	if len(s) != 11 {
		return false
	}
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9, 1, 3}
	sum := 0
	for i := 0; i < 10; i++ {
		sum += int(s[i]-'0') * weights[i]
	}
	check := (10 - sum%10) % 10
	return check == int(s[10]-'0')
}

// nipValid validates Polish NIP (10 digits).
// NipValid checks Polish NIP checksum.
func NipValid(s string) bool { return nipValid(s) }

func nipValid(s string) bool {
	s = digits(s)
	if len(s) != 10 {
		return false
	}
	weights := []int{6, 5, 7, 2, 3, 4, 5, 6, 7}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(s[i]-'0') * weights[i]
	}
	check := sum % 11
	if check == 10 {
		check = 0
	}
	return check == int(s[9]-'0')
}

// regonValid validates Polish REGON (9 or 14 digits).
func regonValid(s string) bool {
	s = digits(s)
	switch len(s) {
	case 9:
		weights := []int{8, 9, 2, 3, 4, 5, 6, 7}
		sum := 0
		for i := 0; i < 8; i++ {
			sum += int(s[i]-'0') * weights[i]
		}
		check := sum % 11
		if check == 10 {
			check = 0
		}
		return check == int(s[8]-'0')
	case 14:
		weights9 := []int{2, 4, 8, 5, 0, 9, 7, 3, 6, 1, 2, 4, 8}
		sum := 0
		for i := 0; i < 13; i++ {
			sum += int(s[i]-'0') * weights9[i]
		}
		check := sum % 11
		if check == 10 {
			check = 0
		}
		return check == int(s[13]-'0')
	default:
		return false
	}
}

// isbnValid validates ISBN-10 or ISBN-13.
func isbnValid(s string) bool {
	s = strings.ToUpper(strings.ReplaceAll(s, "-", ""))
	s = strings.ReplaceAll(s, " ", "")
	if len(s) == 10 {
		sum := 0
		for i := 0; i < 9; i++ {
			if s[i] < '0' || s[i] > '9' {
				return false
			}
			sum += int(s[i]-'0') * (10 - i)
		}
		last := 10
		if s[9] == 'X' {
			last = 10
		} else if s[9] >= '0' && s[9] <= '9' {
			last = int(s[9] - '0')
		} else {
			return false
		}
		return sum%11 == last%11
	}
	if len(s) == 13 {
		sum := 0
		for i := 0; i < 12; i++ {
			if s[i] < '0' || s[i] > '9' {
				return false
			}
			w := 1
			if i%2 == 1 {
				w = 3
			}
			sum += int(s[i]-'0') * w
		}
		check := (10 - sum%10) % 10
		return check == int(s[12]-'0')
	}
	return false
}

func scanMatches(output string, re *regexp.Regexp, validate func(string) bool) bool {
	for _, m := range re.FindAllString(output, -1) {
		if validate(m) {
			return true
		}
	}
	return false
}

// digitsReCache memoises `\b\d{N}\b` regexes; N comes from a small bounded
// set of checksum lengths (PESEL 11, NIP 10, etc).
var digitsReCache sync.Map // int -> *regexp.Regexp

func digitsRegexp(length int) *regexp.Regexp {
	if v, ok := digitsReCache.Load(length); ok {
		return v.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`\b\d{` + strconv.Itoa(length) + `}\b`)
	actual, _ := digitsReCache.LoadOrStore(length, re)
	return actual.(*regexp.Regexp)
}

func scanDigitsMatches(output string, length int, validate func(string) bool) bool {
	re := digitsRegexp(length)
	for _, m := range re.FindAllString(output, -1) {
		if validate(m) {
			return true
		}
	}
	return false
}

func normalizeFieldString(s string) string {
	return strings.TrimSpace(s)
}

func isAlpha2Country(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	_, ok := iso3166Alpha2[code]
	return ok
}

func isISO6391(code string) bool {
	code = strings.ToLower(strings.TrimSpace(code))
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	_, ok := iso6391[code]
	return ok
}

func postalRegex(country string) *regexp.Regexp {
	country = strings.ToUpper(country)
	if re, ok := postalPatterns[country]; ok {
		return re
	}
	return postalPatterns["US"]
}

func containsAnyKeyword(output string, keywords []string) bool {
	lower := strings.ToLower(output)
	for _, k := range keywords {
		k = strings.ToLower(strings.TrimSpace(k))
		if k != "" && strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

func containsAnyPhrase(output string, phrases []string) bool {
	lower := strings.ToLower(output)
	for _, p := range phrases {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" && strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func onlyDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}
