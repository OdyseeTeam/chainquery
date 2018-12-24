package auth

//APIKeys holds the keys for authorized api access
var APIKeys []string

//IsAuthorized checks that the provided key matches the keys provided via the configuration.
func IsAuthorized(key string) bool {
	for _, k := range APIKeys {
		if k == key {
			return true
		}
	}

	return false
}
