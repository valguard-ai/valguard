package validators

import (
	"context"
	"regexp"
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type ValidIBAN struct{}

func (ValidIBAN) Name() string { return "valid_iban" }

func (ValidIBAN) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), ibanValid, "invalid IBAN")
}

type ValidCreditCardLuhn struct{}

func (ValidCreditCardLuhn) Name() string { return "valid_credit_card_luhn" }

func (ValidCreditCardLuhn) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), luhnValid, "invalid card number (Luhn)")
}

type ValidEmailRFC5322 struct{}

func (ValidEmailRFC5322) Name() string { return "valid_email_rfc5322" }

func (ValidEmailRFC5322) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), func(s string) bool {
		return emailRe.MatchString(s)
	}, "invalid email")
}

func (ValidEmailRFC5322) Compile(params map[string]any) CompiledRunner {
	return compileFieldStringCheck(params, func(s string) bool {
		return emailRe.MatchString(s)
	}, "invalid email")
}
