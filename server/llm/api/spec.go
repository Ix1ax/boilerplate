// Package api хранит OpenAPI-спецификацию сервиса llm (openapi.yaml рядом) и встраивает её
// в бинарник: сервис отдаёт её на GET /openapi.yaml, а Swagger в gateway показывает в списке.
package api

import _ "embed"

//go:embed openapi.yaml
var Spec []byte
