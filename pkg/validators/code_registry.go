//go:build !osspublic

package validators

// AllCode returns the 20 Code & Workflow validators (Iteration 5).
func AllCode() []Validator {
	return []Validator{
		ValidSQLSyntax{},
		ValidGraphQLQuery{},
		ValidPythonSyntax{},
		ValidJSSyntax{},
		ValidDockerfile{},
		ValidKubernetesManifest{},
		ValidTerraformHCL{},
		ValidOpenAPISpec{},
		ValidPackageJSON{},
		ValidRequirementsTxt{},
		XMLValidity{},
		XMLXSDSchema{},
		XMLXPathRequired{},
		YAMLValidity{},
		CSVValidity{},
		TOMLValidity{},
		ToolCallWhitelist{},
		ToolCallArgsSchema{},
		FunctionSignatureMatch{},
		StepCountMax{},
	}
}
