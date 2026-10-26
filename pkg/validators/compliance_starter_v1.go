//go:build !osspublic

package validators

import (
	"context"
	"strings"
)

// abaRoutingValid checks US ABA routing transit number (9 digits, mod-10 checksum).
func abaRoutingValid(s string) bool {
	s = digits(s)
	if len(s) != 9 {
		return false
	}
	if s[0:2] == "00" {
		return false
	}
	sum := 3*(int(s[0]-'0')+int(s[3]-'0')+int(s[6]-'0')) +
		7*(int(s[1]-'0')+int(s[4]-'0')+int(s[7]-'0')) +
		(int(s[2]-'0') + int(s[5]-'0') + int(s[8]-'0'))
	return sum%10 == 0
}

// deVatValid validates German USt-IdNr (DE + 9 digits).
func deVatValid(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "DE")
	s = digits(s)
	if len(s) != 9 {
		return false
	}
	weights := []int{8, 7, 6, 5, 4, 3, 2, 1}
	sum := 0
	for i := 0; i < 8; i++ {
		sum += int(s[i]-'0') * weights[i]
	}
	check := 11 - (sum % 11)
	if check == 10 {
		return false
	}
	if check == 11 {
		check = 0
	}
	return check == int(s[8]-'0')
}

type ValidUSRoutingNumber struct{}

func (ValidUSRoutingNumber) Name() string { return "valid_us_routing_number" }

func (ValidUSRoutingNumber) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), abaRoutingValid, "invalid US routing number")
}

type ValidDEVATID struct{}

func (ValidDEVATID) Name() string { return "valid_de_vat_id" }

func (ValidDEVATID) Run(ctx context.Context, output string, params map[string]any) Result {
	return validateJSONFields(ctx, output, stringSliceParam(params["fields"]), deVatValid, "invalid DE VAT ID")
}

func AllStarterV1Compliance() []Validator {
	return []Validator{
		ValidUSRoutingNumber{},
		ValidDEVATID{},
		MaskedCardOnly{},
	}
}
