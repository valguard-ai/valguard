package validators

import (
	"context"
	"regexp"
	"strings"
)

var (
	promptInjectionPatterns = []string{
		`(?i)ignore\s+(all\s+)?(previous|prior)\s+instructions`,
		`(?i)disregard\s+(the\s+)?(above|system)`,
		`(?i)you\s+are\s+now`,
		`(?i)jailbreak`,
	}
	promptInjectionCompiled = mustCompileAll(promptInjectionPatterns)

	secretsDefaultSources = []string{
		`sk-[a-zA-Z0-9]{20,}`,
		`AKIA[0-9A-Z]{16}`,
		`-----BEGIN (?:RSA |EC )?PRIVATE KEY-----`,
		`password\s*[:=]\s*['"][^'"]+['"]`,
	}
	secretsDefaultCompiled = mustCompileAll(secretsDefaultSources)

	urlHostRe = regexp.MustCompile(`https?://([^\s/?:#]+)`)
)

type PromptInjection struct{}

func (PromptInjection) Name() string { return "prompt_injection" }

func (PromptInjection) Run(ctx context.Context, output string, _ map[string]any) Result {
	if m := matchFirstCompiled(output, promptInjectionCompiled, promptInjectionPatterns); m != "" {
		return failResult("error", "prompt injection pattern", map[string]any{"matched": m})
	}
	return passResult("error", "no prompt injection")
}

func (PromptInjection) Compile(_ map[string]any) CompiledRunner {
	return func(_ context.Context, output string) Result {
		if m := matchFirstCompiled(output, promptInjectionCompiled, promptInjectionPatterns); m != "" {
			return failResult("error", "prompt injection pattern", map[string]any{"matched": m})
		}
		return passResult("error", "no prompt injection")
	}
}

type SecretsDetection struct{ patternValidator }

func (SecretsDetection) Name() string { return "secrets_detection" }

func (v SecretsDetection) Run(ctx context.Context, output string, params map[string]any) Result {
	regs, err := resolveSecretsPatterns(params)
	if err != nil {
		return failResult("error", "no valid patterns in params", map[string]any{"error": err.Error()})
	}
	return secretsDetectionRun(output, regs)
}

func (SecretsDetection) Compile(params map[string]any) CompiledRunner {
	regs, err := resolveSecretsPatterns(params)
	if err != nil {
		msg := err.Error()
		return func(_ context.Context, _ string) Result {
			return failResult("error", "no valid patterns in params", map[string]any{"error": msg})
		}
	}
	return func(_ context.Context, output string) Result {
		return secretsDetectionRun(output, regs)
	}
}

func resolveSecretsPatterns(params map[string]any) ([]*regexp.Regexp, error) {
	custom := stringSliceParam(params["patterns"])
	if len(custom) == 0 {
		return secretsDefaultCompiled, nil
	}
	return compileCustomRegexPatterns(custom)
}

func secretsDetectionRun(output string, regs []*regexp.Regexp) Result {
	for _, re := range regs {
		if re.MatchString(output) {
			return failResult("error", "pattern matched", map[string]any{"matched": re.String()})
		}
	}
	return passResult("error", "ok")
}

type InternalNetworkURL struct{}

func (InternalNetworkURL) Name() string { return "internal_network_url" }

func (InternalNetworkURL) Run(ctx context.Context, output string, params map[string]any) Result {
	blocked := stringSliceParam(params["blocked_hosts"])
	if len(blocked) == 0 {
		blocked = []string{"localhost", "127.0.0.1", "10.0.0.0/8", "192.168.0.0/16", "169.254.0.0/16"}
	}
	for _, m := range urlHostRe.FindAllStringSubmatch(output, -1) {
		host := strings.ToLower(m[1])
		if hostBlocked(host, blocked) {
			return failResult("error", "internal or private URL", map[string]any{"host": host})
		}
	}
	return passResult("error", "no internal URLs")
}

func hostBlocked(host string, blocked []string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	for _, b := range blocked {
		b = strings.ToLower(strings.TrimSpace(b))
		if b == "" {
			continue
		}
		if strings.Contains(b, "/") {
			if cidrMatch(host, b) {
				return true
			}
			continue
		}
		if host == b || strings.HasSuffix(host, "."+b) {
			return true
		}
	}
	return false
}

func cidrMatch(host, cidr string) bool {
	parts := strings.SplitN(cidr, "/", 2)
	if len(parts) != 2 {
		return host == cidr
	}
	prefix := parts[0]
	if host == prefix {
		return true
	}
	if strings.HasPrefix(prefix, "10.") && strings.HasPrefix(host, "10.") {
		return true
	}
	if strings.HasPrefix(prefix, "192.168.") && strings.HasPrefix(host, "192.168.") {
		return true
	}
	if strings.HasPrefix(prefix, "169.254.") && strings.HasPrefix(host, "169.254.") {
		return true
	}
	if strings.HasPrefix(prefix, "172.") && strings.HasPrefix(host, "172.") {
		return true
	}
	return false
}
