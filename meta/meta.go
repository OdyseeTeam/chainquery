package meta

// version and commitMsg get filled in using -ldflags when the binary gets built with /scripts/build.sh
var version string
var versionLong string
var commitMsg string

// GetVersion returns the version of the application. If it is tagged it will be the semver, otherwise it will contain
// the number of commits since the last one, the short hash of the commit and whether or not the directory was dirty
// at build time.
func GetVersion() string {
	if version != "" {
		return version
	}

	return "unknown"
}

// GetVersionLong returns what GetVersion returns but will always return the long version.
func GetVersionLong() string {
	if versionLong != "" {
		return versionLong
	}

	return "unknown"
}

// GetCommitMessage returns the commit message of the commit used to build the binary.
func GetCommitMessage() string {
	if commitMsg != "" {
		return commitMsg
	}

	return "unknown"
}
