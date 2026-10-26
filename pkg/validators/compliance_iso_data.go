//go:build !osspublic

package validators

import "regexp"

var iso3166Alpha2 = map[string]struct{}{
	"US": {}, "GB": {}, "DE": {}, "FR": {}, "PL": {}, "CH": {}, "AT": {}, "BE": {}, "NL": {},
	"ES": {}, "IT": {}, "CA": {}, "AU": {}, "JP": {}, "CN": {}, "IN": {}, "BR": {}, "MX": {},
}

var iso6391 = map[string]struct{}{
	"en": {}, "de": {}, "fr": {}, "pl": {}, "es": {}, "it": {}, "pt": {}, "nl": {}, "cs": {},
	"sv": {}, "da": {}, "no": {}, "fi": {}, "ja": {}, "zh": {}, "ru": {}, "uk": {}, "ar": {},
}

var postalPatterns = map[string]*regexp.Regexp{
	"PL": regexp.MustCompile(`^\d{2}-\d{3}$`),
	"US": regexp.MustCompile(`^\d{5}(-\d{4})?$`),
	"GB": regexp.MustCompile(`^[A-Z]{1,2}\d[A-Z\d]?\s*\d[A-Z]{2}$`),
	"DE": regexp.MustCompile(`^\d{5}$`),
	"FR": regexp.MustCompile(`^\d{5}$`),
}
