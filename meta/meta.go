package meta

// version and commitMsg get filled in using -ldflags when the binary gets built with /scripts/build.sh
var semVersion string
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

// GetSemVersion returns the Semantic Version of the Chainquery Application. This field is set on deployment
// with the tag vx.x.x
func GetSemVersion() string {
	if semVersion != "" {
		return semVersion
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
