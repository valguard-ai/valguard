package validators

import (
	"os"

	"gopkg.in/yaml.v3"
)

type CatalogEntry struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Enabled     bool           `yaml:"enabled"`
	Priority    int            `yaml:"priority"`
	Severity    string         `yaml:"severity"`
	Params      map[string]any `yaml:"params"`
	OnFail      string         `yaml:"on_fail"`
	Category    string         `yaml:"category"`
}

type Catalog struct {
	Validators []CatalogEntry `yaml:"validators"`
}

func LoadCatalog(paths ...string) (Catalog, error) {
	var merged Catalog
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return merged, err
		}
		var c Catalog
		if err := yaml.Unmarshal(b, &c); err != nil {
			return merged, err
		}
		merged.Validators = append(merged.Validators, c.Validators...)
	}
	return merged, nil
}

func DefaultCatalogPaths() []string {
	prefixes := []string{
		"../../../../config/",
		"../../../config/",
		"../../config/",
		"config/",
	}
	files := []string{
		"validators.master.yaml",
		"validators.extended.yaml",
		"validators.phase_b.yaml",
	}
	var paths []string
	for _, p := range prefixes {
		for _, f := range files {
			paths = append(paths, p+f)
		}
	}
	return paths
}
