package docsdata

import _ "embed"

// OpenAPIYAML is the operator API OpenAPI 3 document (see docs/openapi.yaml).
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
