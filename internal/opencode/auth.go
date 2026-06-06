package opencode

const providerID = "voban"

// writeVobanAuth merges the Voban API key into opencode's auth.json, preserving
// any other providers' credentials. The file is written with 0600 permissions
// because it holds a secret.
func writeVobanAuth(path, apiKey string) error {
	auth, err := readJSONObject(path)
	if err != nil {
		return err
	}
	auth[providerID] = map[string]any{
		"type": "api",
		"key":  apiKey,
	}
	return writeJSONObject(path, auth, 0o600)
}
