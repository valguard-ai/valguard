//go:build !osspublic

package validators

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// countryCodeWordReCache memoises `\b<CODE>\b` regexes used to test for
// forbidden ISO country codes without false positives on common words.
var countryCodeWordReCache sync.Map // string -> *regexp.Regexp

func countryCodeWordRegexp(code string) *regexp.Regexp {
	if v, ok := countryCodeWordReCache.Load(code); ok {
		return v.(*regexp.Regexp)
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(code) + `\b`)
	actual, _ := countryCodeWordReCache.LoadOrStore(code, re)
	return actual.(*regexp.Regexp)
}

// E.164 calling codes for allowed_countries checks on phone fields (subset).
var phonePrefixByCountry = map[string][]string{
	"US": {"+1"},
	"CA": {"+1"},
	"GB": {"+44"},
	"DE": {"+49"},
	"FR": {"+33"},
	"PL": {"+48"},
	"CH": {"+41"},
	"AT": {"+43"},
	"NL": {"+31"},
	"ES": {"+34"},
	"IT": {"+39"},
	"AU": {"+61"},
	"JP": {"+81"},
	"IN": {"+91"},
	"BR": {"+55"},
	"MX": {"+52"},
}

// GeoCountryWhitelist ensures country fields only use allowed ISO 3166-1 alpha-2 codes.
type GeoCountryWhitelist struct{}

func (GeoCountryWhitelist) Name() string { return "geo_country_whitelist" }

func (GeoCountryWhitelist) Run(ctx context.Context, output string, params map[string]any) Result {
	allowed := normalizeCountryCodes(stringSliceParam(params["allowed_countries"]))
	if len(allowed) == 0 {
		return skipResult("no allowed_countries configured")
	}
	allowSet := make(map[string]struct{}, len(allowed))
	for _, c := range allowed {
		allowSet[c] = struct{}{}
	}

	fields := stringSliceParam(params["fields"])
	if len(fields) > 0 {
		data, err := parseJSONObjectCtx(ctx, output)
		if err != nil {
			return failResult("error", "cannot parse JSON for geo_country_whitelist", nil)
		}
		var bad []string
		for _, f := range fields {
			val, ok := data[f]
			if !ok {
				continue
			}
			code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(val)))
			if !isAlpha2Format(code) {
				bad = append(bad, f+": invalid country code format")
				continue
			}
			if _, ok := allowSet[code]; !ok {
				bad = append(bad, f+": "+code)
			}
		}
		if len(bad) > 0 {
			return failResult("error", "country not in allowed list", map[string]any{"violations": bad})
		}
		return passResult("error", "countries in whitelist")
	}
	return skipResult("configure fields for country codes in JSON output")
}

// GeoCoordinatesInBounds validates lat/lon JSON fields stay within a bounding box.
type GeoCoordinatesInBounds struct{}

func (GeoCoordinatesInBounds) Name() string { return "geo_coordinates_in_bounds" }

func (GeoCoordinatesInBounds) Run(ctx context.Context, output string, params map[string]any) Result {
	latField, _ := params["lat_field"].(string)
	lonField, _ := params["lon_field"].(string)
	if latField == "" {
		latField = "lat"
	}
	if lonField == "" {
		lonField = "lon"
	}
	minLat := floatParam(params["min_lat"], -90)
	maxLat := floatParam(params["max_lat"], 90)
	minLon := floatParam(params["min_lon"], -180)
	maxLon := floatParam(params["max_lon"], 180)

	data, err := parseJSONObjectCtx(ctx, output)
	if err != nil {
		return failResult("error", "cannot parse JSON for coordinates", nil)
	}
	lat, latOk := floatFromAny(data[latField])
	lon, lonOk := floatFromAny(data[lonField])
	if !latOk || !lonOk {
		return failResult("error", "missing or invalid lat/lon fields", map[string]any{
			"lat_field": latField,
			"lon_field": lonField,
		})
	}
	if lat < minLat || lat > maxLat || lon < minLon || lon > maxLon {
		return failResult("error", "coordinates outside allowed bounds", map[string]any{
			"lat": lat, "lon": lon,
			"min_lat": minLat, "max_lat": maxLat,
			"min_lon": minLon, "max_lon": maxLon,
		})
	}
	return passResult("error", "coordinates within bounds")
}

// GeoForbiddenCountries blocks forbidden ISO country codes in output (fields or full-text scan).
type GeoForbiddenCountries struct{}

func (GeoForbiddenCountries) Name() string { return "geo_forbidden_countries" }

func (GeoForbiddenCountries) Run(ctx context.Context, output string, params map[string]any) Result {
	forbidden := normalizeCountryCodes(stringSliceParam(params["forbidden_countries"]))
	if len(forbidden) == 0 {
		return skipResult("no forbidden_countries configured")
	}
	forbidSet := make(map[string]struct{}, len(forbidden))
	for _, c := range forbidden {
		forbidSet[c] = struct{}{}
	}

	fields := stringSliceParam(params["fields"])
	if len(fields) > 0 {
		data, err := parseJSONObjectCtx(ctx, output)
		if err != nil {
			return failResult("error", "cannot parse JSON for geo_forbidden_countries", nil)
		}
		var hit []string
		for _, f := range fields {
			val, ok := data[f]
			if !ok {
				continue
			}
			code := strings.ToUpper(strings.TrimSpace(fmt.Sprint(val)))
			if _, banned := forbidSet[code]; banned {
				hit = append(hit, f+": "+code)
			}
		}
		if len(hit) > 0 {
			return failResult("error", "forbidden country present", map[string]any{"violations": hit})
		}
		return passResult("error", "no forbidden countries in fields")
	}

	upper := strings.ToUpper(output)
	var hit []string
	for code := range forbidSet {
		if strings.Contains(upper, code) {
			// avoid false positives on common words - require word boundary
			if countryCodeWordRegexp(code).MatchString(upper) {
				hit = append(hit, code)
			}
		}
	}
	if len(hit) > 0 {
		return failResult("error", "forbidden country code in output", map[string]any{"codes": dedupeStrings(hit)})
	}
	return passResult("error", "no forbidden countries detected")
}

func normalizeCountryCodes(codes []string) []string {
	var out []string
	for _, c := range codes {
		c = strings.ToUpper(strings.TrimSpace(c))
		if isAlpha2Format(c) {
			out = append(out, c)
		}
	}
	return out
}

func isAlpha2Format(code string) bool {
	if len(code) != 2 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

func floatParam(v any, def float64) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f
		}
	}
	return def
}

func floatFromAny(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, !math.IsNaN(x)
	case int:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil && !math.IsNaN(f)
	}
	return 0, false
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func phoneMatchesAllowedCountries(number string, allowed []string) bool {
	number = strings.TrimSpace(number)
	for _, cc := range allowed {
		for _, prefix := range phonePrefixByCountry[cc] {
			if strings.HasPrefix(number, prefix) {
				return true
			}
		}
	}
	return false
}
