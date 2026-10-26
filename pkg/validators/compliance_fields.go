//go:build !osspublic

package validators

import (
	"context"
	"strings"
)

type ValidISBN struct{}

func (ValidISBN) Name() string { return "valid_isbn" }

func (ValidISBN) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), isbnValid, "invalid ISBN")
}

type ValidPESEL struct{}

func (ValidPESEL) Name() string { return "valid_pesel" }

func (ValidPESEL) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), peselValid, "invalid PESEL")
}

type ValidNIP struct{}

func (ValidNIP) Name() string { return "valid_nip" }

func (ValidNIP) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), nipValid, "invalid NIP")
}

type ValidREGON struct{}

func (ValidREGON) Name() string { return "valid_regon" }

func (ValidREGON) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), regonValid, "invalid REGON")
}

type ValidPostalCode struct{}

func (ValidPostalCode) Name() string { return "valid_postal_code" }

func (ValidPostalCode) Run(ctx context.Context, output string, params map[string]any) Result {
	country, _ := params["country"].(string)
	if country == "" {
		country = "PL"
	}
	re := postalRegex(country)
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), func(s string) bool {
		return re.MatchString(strings.TrimSpace(s))
	}, "invalid postal code")
}
