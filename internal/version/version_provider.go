package version

type VersionProvider struct {
	version string
}

func (v *VersionProvider) String() string {
	return v.version
}

func NewVersionProvider(version string) *VersionProvider {
	return &VersionProvider{version: version}
}
