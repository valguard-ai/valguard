//go:build !osspublic

package validators

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type ValidSQLSyntax struct{}

func (ValidSQLSyntax) Name() string { return "valid_sql_syntax" }

var (
	sqlStatementRe   = regexp.MustCompile(`(?i)\b(SELECT|INSERT|UPDATE|DELETE|WITH|CREATE|ALTER|DROP|TRUNCATE|MERGE)\b`)
	graphqlOpRe      = regexp.MustCompile(`(?i)\b(query|mutation|subscription)\b`)
	pythonStmtKwRe   = regexp.MustCompile(`^\s*(def|class|if|elif|else|for|while|try|except|finally|with|match|case)\b`)
	dockerfileInstRe = regexp.MustCompile(`(?i)^(FROM|RUN|CMD|LABEL|EXPOSE|ENV|ADD|COPY|ENTRYPOINT|VOLUME|USER|WORKDIR|ARG|ONBUILD|STOPSIGNAL|HEALTHCHECK|SHELL)\b`)
	terraformBlockRe = regexp.MustCompile(`(?m)(resource|data|module|provider|terraform|variable|output)\s+"`)
	requirementsRe   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\-]*(\[[^\]]+\])?([<>=!~]=?|===)\s*.+$|^[A-Za-z0-9][A-Za-z0-9._\-]*$|^-[rR]\s`)
)

func (ValidSQLSyntax) Run(ctx context.Context, output string, _ map[string]any) Result {
	sql := stripSQLComments(strings.TrimSpace(output))
	if sql == "" {
		return failResult("error", "SQL output is empty", nil)
	}
	if !sqlStatementRe.MatchString(sql) {
		return failResult("error", "SQL does not contain a recognizable statement", nil)
	}
	if !balancedDelimiters(sql, [2]byte{'(', ')'}) {
		return failResult("error", "unbalanced parentheses in SQL", nil)
	}
	return passResult("error", "SQL syntax looks valid")
}

type ValidGraphQLQuery struct{}

func (ValidGraphQLQuery) Name() string { return "valid_graphql_query" }

func (ValidGraphQLQuery) Run(ctx context.Context, output string, _ map[string]any) Result {
	q := strings.TrimSpace(output)
	if q == "" {
		return failResult("error", "GraphQL query is empty", nil)
	}
	if !balancedDelimiters(q, [2]byte{'{', '}'}, [2]byte{'(', ')'}, [2]byte{'[', ']'}) {
		return failResult("error", "unbalanced braces in GraphQL", nil)
	}
	if !graphqlOpRe.MatchString(q) && !strings.Contains(q, "{") {
		return failResult("error", "GraphQL must include an operation or selection set", nil)
	}
	return passResult("error", "GraphQL syntax looks valid")
}

type ValidPythonSyntax struct{}

func (ValidPythonSyntax) Name() string { return "valid_python_syntax" }

func (ValidPythonSyntax) Run(ctx context.Context, output string, params map[string]any) Result {
	blocks := extractCodeBlocksParam(output, params)
	if len(blocks) == 0 {
		return failResult("error", "no Python code to validate", nil)
	}
	for i, block := range blocks {
		if err := checkPythonBlock(block); err != nil {
			return failResult("error", fmt.Sprintf("Python block %d: %s", i+1, err.Error()), nil)
		}
	}
	return passResult("error", "Python syntax looks valid")
}

func checkPythonBlock(block string) error {
	if !balancedDelimiters(block, [2]byte{'(', ')'}, [2]byte{'[', ']'}, [2]byte{'{', '}'}) {
		return fmt.Errorf("unbalanced delimiters")
	}
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if pythonStmtKwRe.MatchString(trim) {
			if !strings.HasSuffix(strings.Split(trim, "#")[0], ":") {
				return fmt.Errorf("expected ':' at end of line: %s", trim)
			}
		}
	}
	return nil
}

type ValidJSSyntax struct{}

func (ValidJSSyntax) Name() string { return "valid_js_syntax" }

func (ValidJSSyntax) Run(ctx context.Context, output string, params map[string]any) Result {
	blocks := extractCodeBlocksParam(output, params)
	if len(blocks) == 0 {
		return failResult("error", "no JavaScript to validate", nil)
	}
	for i, block := range blocks {
		trim := strings.TrimSpace(block)
		if trim == "" {
			return failResult("error", "empty JS block", nil)
		}
		if strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "[") {
			var v any
			if err := json.Unmarshal([]byte(trim), &v); err != nil {
				return failResult("error", fmt.Sprintf("JS/JSON block %d: %s", i+1, err.Error()), nil)
			}
			continue
		}
		if !balancedDelimiters(trim, [2]byte{'(', ')'}, [2]byte{'[', ']'}, [2]byte{'{', '}'}) {
			return failResult("error", fmt.Sprintf("JS block %d: unbalanced delimiters", i+1), nil)
		}
	}
	return passResult("error", "JavaScript syntax looks valid")
}

type ValidDockerfile struct{}

func (ValidDockerfile) Name() string { return "valid_dockerfile" }

func (ValidDockerfile) Run(ctx context.Context, output string, _ map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "Dockerfile is empty", nil)
	}
	lines := strings.Split(content, "\n")
	hasFrom := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(line), "FROM ") {
			hasFrom = true
		}
		if !dockerfileInstRe.MatchString(line) && !strings.HasPrefix(line, "#") {
			return failResult("error", "unknown Dockerfile instruction: "+line, nil)
		}
	}
	if !hasFrom {
		return failResult("error", "Dockerfile must contain a FROM instruction", nil)
	}
	return passResult("error", "Dockerfile looks valid")
}

type ValidKubernetesManifest struct{}

func (ValidKubernetesManifest) Name() string { return "valid_kubernetes_manifest" }

func (ValidKubernetesManifest) Run(ctx context.Context, output string, params map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "manifest is empty", nil)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return failResult("error", "invalid YAML manifest", map[string]any{"error": err.Error()})
	}
	apiVersion, _ := doc["apiVersion"].(string)
	kind, _ := doc["kind"].(string)
	if apiVersion == "" || kind == "" {
		return failResult("error", "Kubernetes manifest requires apiVersion and kind", nil)
	}
	if kinds := stringSliceParam(params["resource_kinds"]); len(kinds) > 0 {
		ok := false
		for _, k := range kinds {
			if strings.EqualFold(kind, k) {
				ok = true
				break
			}
		}
		if !ok {
			return failResult("error", "kind not in resource_kinds whitelist", map[string]any{"kind": kind})
		}
	}
	return passResult("error", "Kubernetes manifest looks valid")
}

type ValidTerraformHCL struct{}

func (ValidTerraformHCL) Name() string { return "valid_terraform_hcl" }

func (ValidTerraformHCL) Run(ctx context.Context, output string, _ map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "HCL is empty", nil)
	}
	if !balancedDelimiters(content, [2]byte{'{', '}'}, [2]byte{'[', ']'}, [2]byte{'(', ')'}) {
		return failResult("error", "unbalanced delimiters in HCL", nil)
	}
	if !terraformBlockRe.MatchString(content) {
		return failResult("error", "HCL must contain a Terraform block declaration", nil)
	}
	return passResult("error", "Terraform HCL looks valid")
}

type ValidOpenAPISpec struct{}

func (ValidOpenAPISpec) Name() string { return "valid_openapi_spec" }

func (ValidOpenAPISpec) Run(ctx context.Context, output string, params map[string]any) Result {
	expectVersion, _ := params["version"].(string)
	return validateOpenAPISpec(output, expectVersion)
}

type ValidPackageJSON struct{}

func (ValidPackageJSON) Name() string { return "valid_package_json" }

func (ValidPackageJSON) Run(ctx context.Context, output string, _ map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "package.json is empty", nil)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return failResult("error", "invalid package.json JSON", map[string]any{"error": err.Error()})
	}
	if _, ok := doc["name"]; !ok {
		return failResult("error", "package.json must include name", nil)
	}
	if _, ok := doc["version"]; !ok {
		return failResult("error", "package.json must include version", nil)
	}
	return passResult("error", "package.json looks valid")
}

type ValidRequirementsTxt struct{}

func (ValidRequirementsTxt) Name() string { return "valid_requirements_txt" }

func (ValidRequirementsTxt) Run(ctx context.Context, output string, _ map[string]any) Result {
	content := strings.TrimSpace(output)
	if content == "" {
		return failResult("error", "requirements.txt is empty", nil)
	}
	for i, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "-") && requirementsRe.MatchString(line) {
			continue
		}
		if !requirementsRe.MatchString(line) {
			return failResult("error", fmt.Sprintf("invalid requirements line %d", i+1), map[string]any{"line": line})
		}
	}
	return passResult("error", "requirements.txt looks valid")
}
