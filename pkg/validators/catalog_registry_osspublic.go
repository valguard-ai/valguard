//go:build osspublic

package validators

// NewCatalogRegistry builds the OSS-public registry (40 MVP rules only).
func NewCatalogRegistry() *Registry {
	return NewRegistry(AllMVPValidators()...)
}
