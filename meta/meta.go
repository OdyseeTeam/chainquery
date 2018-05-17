package meta

// version and commitMsg get filled in using -ldflags when the binary gets built
var version string
var versionLong string
var commitMsg string

func GetVersion() string {
	if version != "" {
		return version
	}

	return "unknown"
}

func GetVersionLong() string {
	if versionLong != "" {
		return versionLong
	}

	return "unknown"
}

func GetCommitMessage() string {
	if commitMsg != "" {
		return commitMsg
	}

	return "unknown"
}
