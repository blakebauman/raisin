package openapi

import _ "embed"

// SpecYAML is the Raisin Email API OpenAPI 3 document.
//
//go:embed openapi.yaml
var SpecYAML []byte
