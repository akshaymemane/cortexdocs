package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/akshaymemane/cortexdocs/internal/model"
	"github.com/akshaymemane/cortexdocs/internal/parser"
)

func BuildSpec(sourcePath, name string, parsed parser.ParseResult) model.Spec {
	if name == "" {
		name = "CortexDocs"
	}
	spec := model.Spec{
		Name:        name,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SourcePath:  sourcePath,
		Files:       parsed.Files,
		Warnings:    parsed.Warnings,
	}

	functions := dedupeFunctions(parsed.Functions)
	structs := dedupeStructs(parsed.Structs)
	enums := dedupeEnums(parsed.Enums)

	spec.Functions = make([]model.Function, 0, len(functions))
	endpoints := make([]model.Endpoint, 0, len(functions)+len(parsed.Endpoints))

	for _, fn := range functions {
		function := model.Function{
			ID:          slugify(fn.Name + "-" + filepath.Base(fn.File)),
			Name:        fn.Name,
			Description: fallbackDescription(fn.Description, "Undocumented function."),
			Signature:   fn.Signature,
			ReturnType:  fn.ReturnType,
			File:        fn.File,
			Line:        fn.Line,
			Params:      mapParams(fn.Params),
			Deprecated:  fn.Deprecated,
			Example:     fn.Example,
		}
		spec.Functions = append(spec.Functions, function)

		if fn.Route != nil {
			endpoints = append(endpoints, model.Endpoint{
				ID:          slugify(fn.Route.Method + "-" + fn.Route.Path),
				Name:        fn.Name,
				Method:      fn.Route.Method,
				Path:        fn.Route.Path,
				Description: fallbackDescription(fn.Description, "No description provided."),
				Signature:   fn.Signature,
				ReturnType:  fn.ReturnType,
				File:        fn.File,
				Line:        fn.Line,
				Params:      mapParams(fn.Params),
				Responses:   mapResponses(fn.Responses),
				Source:      "docblock",
				Deprecated:  fn.Deprecated,
				Example:     fn.Example,
			})
		}
	}

	for _, endpoint := range parsed.Endpoints {
		src := endpoint.Source
		if src == "" {
			src = "heuristic"
		}
		endpoints = append(endpoints, model.Endpoint{
			ID:          slugify(endpoint.Method + "-" + endpoint.Path + "-" + endpoint.Name),
			Name:        endpoint.Name,
			Method:      endpoint.Method,
			Path:        endpoint.Path,
			Description: fallbackDescription(endpoint.Description, "Heuristically extracted from source."),
			Signature:   endpoint.Signature,
			ReturnType:  endpoint.ReturnType,
			File:        endpoint.File,
			Line:        endpoint.Line,
			Params:      mapParams(endpoint.Params),
			Responses:   mapResponses(endpoint.Responses),
			Source:      src,
			Deprecated:  endpoint.Deprecated,
			Example:     endpoint.Example,
		})
	}
	spec.Endpoints = dedupeEndpoints(endpoints)

	spec.Structs = make([]model.Struct, 0, len(structs))
	for _, item := range structs {
		fields := make([]model.Field, 0, len(item.Fields))
		for _, field := range item.Fields {
			fields = append(fields, model.Field{
				Name:        field.Name,
				Type:        field.Type,
				Description: field.Description,
			})
		}
		spec.Structs = append(spec.Structs, model.Struct{
			ID:          slugify("struct-" + item.Name),
			Name:        item.Name,
			Description: fallbackDescription(item.Description, "C struct extracted from source."),
			File:        item.File,
			Line:        item.Line,
			Fields:      fields,
		})
	}

	spec.Enums = make([]model.Enum, 0, len(enums))
	for _, item := range enums {
		values := make([]model.EnumValue, 0, len(item.Values))
		for _, value := range item.Values {
			values = append(values, model.EnumValue{
				Name:        value.Name,
				Description: value.Description,
			})
		}
		spec.Enums = append(spec.Enums, model.Enum{
			ID:          slugify("enum-" + item.Name),
			Name:        item.Name,
			Description: fallbackDescription(item.Description, "C enum extracted from source."),
			File:        item.File,
			Line:        item.Line,
			Values:      values,
		})
	}

	slices.SortFunc(spec.Functions, func(a, b model.Function) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(spec.Endpoints, func(a, b model.Endpoint) int {
		if compare := strings.Compare(a.Path, b.Path); compare != 0 {
			return compare
		}
		return strings.Compare(a.Method, b.Method)
	})
	slices.SortFunc(spec.Structs, func(a, b model.Struct) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(spec.Enums, func(a, b model.Enum) int { return strings.Compare(a.Name, b.Name) })

	spec.Summary = model.Summary{
		EndpointCount: len(spec.Endpoints),
		FunctionCount: len(spec.Functions),
		StructCount:   len(spec.Structs),
		EnumCount:     len(spec.Enums),
	}

	return spec
}

func WriteJSON(spec model.Spec, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	payload, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal api json: %w", err)
	}

	if err := os.WriteFile(outputPath, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	return nil
}

func mapParams(params []parser.ParamDoc) []model.Parameter {
	items := make([]model.Parameter, 0, len(params))
	for _, param := range params {
		items = append(items, model.Parameter{
			Name:        param.Name,
			Type:        param.Type,
			Description: param.Description,
			Direction:   param.Direction,
		})
	}
	return items
}

func mapResponses(responses []parser.ResponseDoc) []model.Response {
	items := make([]model.Response, 0, len(responses))
	for _, response := range responses {
		items = append(items, model.Response{
			Status:      response.Status,
			Type:        response.Type,
			Description: response.Description,
		})
	}
	return items
}

func dedupeFunctions(items []parser.FunctionDoc) []parser.FunctionDoc {
	best := map[string]parser.FunctionDoc{}
	for _, item := range items {
		key := item.Name + "|" + item.Signature
		existing, ok := best[key]
		if !ok || functionScore(item) > functionScore(existing) {
			best[key] = item
		}
	}

	keys := make([]string, 0, len(best))
	for key := range best {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	result := make([]parser.FunctionDoc, 0, len(keys))
	for _, key := range keys {
		result = append(result, best[key])
	}
	return result
}

func dedupeStructs(items []parser.StructDoc) []parser.StructDoc {
	best := map[string]parser.StructDoc{}
	for _, item := range items {
		existing, ok := best[item.Name]
		if !ok || len(item.Fields) > len(existing.Fields) {
			best[item.Name] = item
		}
	}
	keys := make([]string, 0, len(best))
	for key := range best {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	result := make([]parser.StructDoc, 0, len(keys))
	for _, key := range keys {
		result = append(result, best[key])
	}
	return result
}

func dedupeEnums(items []parser.EnumDoc) []parser.EnumDoc {
	best := map[string]parser.EnumDoc{}
	for _, item := range items {
		existing, ok := best[item.Name]
		if !ok || len(item.Values) > len(existing.Values) {
			best[item.Name] = item
		}
	}
	keys := make([]string, 0, len(best))
	for key := range best {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	result := make([]parser.EnumDoc, 0, len(keys))
	for _, key := range keys {
		result = append(result, best[key])
	}
	return result
}

func dedupeEndpoints(items []model.Endpoint) []model.Endpoint {
	best := map[string]model.Endpoint{}
	for _, item := range items {
		key := item.Method + "|" + item.Path
		existing, ok := best[key]
		if !ok || endpointScore(item) > endpointScore(existing) {
			best[key] = item
		}
	}
	keys := make([]string, 0, len(best))
	for key := range best {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	result := make([]model.Endpoint, 0, len(keys))
	for _, key := range keys {
		result = append(result, best[key])
	}
	return result
}

func functionScore(item parser.FunctionDoc) int {
	score := len(item.Params) + len(item.Responses)
	if item.Description != "" {
		score += 5
	}
	if item.Route != nil {
		score += 10
	}
	return score
}

func endpointScore(item model.Endpoint) int {
	score := len(item.Params) + len(item.Responses)
	if item.Description != "" {
		score += 5
	}
	if item.Signature != "" {
		score += 3
	}
	return score
}

func fallbackDescription(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func slugify(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-", ":", "-", "|", "-", "*", "", "(", "", ")", "")
	value = replacer.Replace(value)

	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
