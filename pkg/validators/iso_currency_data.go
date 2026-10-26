package validators

import "strings"

// MVP currency codes (subset; full ISO list lives in enterprise catalog data).
var iso4217 = map[string]struct{}{
	"USD": {}, "EUR": {}, "GBP": {}, "PLN": {}, "CHF": {}, "JPY": {}, "CAD": {}, "AUD": {},
	"CZK": {}, "SEK": {}, "NOK": {}, "DKK": {}, "HUF": {}, "RON": {}, "BGN": {}, "HRK": {},
}

func isISO4217(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	_, ok := iso4217[code]
	return ok
}
