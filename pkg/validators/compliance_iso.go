//go:build !osspublic

package validators

import "context"

type ValidISOCurrency struct{}

func (ValidISOCurrency) Name() string { return "valid_iso_currency" }

func (ValidISOCurrency) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), isISO4217, "invalid ISO 4217 currency")
}

func (ValidISOCurrency) Compile(params map[string]any) CompiledRunner {
	return compileFieldStringCheck(params, isISO4217, "invalid ISO 4217 currency")
}

type ValidISOCountry struct{}

func (ValidISOCountry) Name() string { return "valid_iso_country" }

func (ValidISOCountry) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), isAlpha2Country, "invalid ISO 3166 country")
}

func (ValidISOCountry) Compile(params map[string]any) CompiledRunner {
	return compileFieldStringCheck(params, isAlpha2Country, "invalid ISO 3166 country")
}

type ValidISOLanguage struct{}

func (ValidISOLanguage) Name() string { return "valid_iso_language" }

func (ValidISOLanguage) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), isISO6391, "invalid ISO 639-1 language")
}

func (ValidISOLanguage) Compile(params map[string]any) CompiledRunner {
	return compileFieldStringCheck(params, isISO6391, "invalid ISO 639-1 language")
}
