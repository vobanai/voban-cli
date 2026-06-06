package opencode

// StoredKey returns the Voban API key previously written to opencode's auth.json,
// if present. It lets the models and status commands reuse the key from a prior
// configure run without prompting again.
func StoredKey() (string, bool, error) {
	resolver, err := newPathResolver()
	if err != nil {
		return "", false, err
	}
	return storedKey(resolver.authFile())
}

func storedKey(authPath string) (string, bool, error) {
	auth, err := readJSONObject(authPath)
	if err != nil {
		return "", false, err
	}
	entry, ok := auth[providerID].(map[string]any)
	if !ok {
		return "", false, nil
	}
	key, ok := entry["key"].(string)
	if !ok || key == "" {
		return "", false, nil
	}
	return key, true, nil
}
