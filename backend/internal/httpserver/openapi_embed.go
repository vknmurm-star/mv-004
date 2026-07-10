package httpserver

import _ "embed"

// openAPIBytes is embedded at build time so the spec is served regardless of
// the process working directory (critical for the distroless Docker image,
// where there is no source tree on disk to os.ReadFile from).
//
//go:embed openapi.yaml
var openAPIBytes []byte

// openAPIDocument returns the bundled OpenAPI spec.
func openAPIDocument() ([]byte, error) {
	return openAPIBytes, nil
}
