//go:build osspublic

package validators

import (
	"regexp"
	"strconv"
	"strings"
)

var digitsOnly = regexp.MustCompile(`\D`)

func digits(s string) string {
	return digitsOnly.ReplaceAllString(s, "")
}

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
		return false
	}
	return check == int(s[9]-'0')
}
