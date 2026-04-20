package parser

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	cStringLiteralPattern        = regexp.MustCompile(`"((?:\\.|[^"\\])*)"`)
	httpMethodPattern            = regexp.MustCompile(`(?:^|[^A-Z])(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD|ANY)(?:[^A-Z]|$)`)
	h2oRegisterHandlerPattern    = regexp.MustCompile(`\bregister_handler\s*\([^,]+,\s*"((?:\\.|[^"\\])*)"\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)`)
	h2oConfigRegisterPathPattern = regexp.MustCompile(`\bh2o_config_register_path\s*\([^,]+,\s*"((?:\\.|[^"\\])*)"\s*,`)
)

type endpointCandidate struct {
	EndpointDoc
	Confidence int
}

func readSourceLines(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n"), nil
}

func inferEndpoints(path string, lines []string, comments []commentBlock, functions []FunctionDoc) []EndpointDoc {
	if len(lines) == 0 {
		return nil
	}

	candidates := map[string]endpointCandidate{}
	addEndpointCandidates(candidates, inferH2ORegistrationEndpoints(path, lines, comments, functions))
	for index := range lines {
		if isH2ORegistrationLine(lines[index]) {
			continue
		}
		window := buildWindow(lines, index, 0, 2)
		paths := extractPaths(window)
		if len(paths) == 0 {
			continue
		}

		contextScore := routeContextScore(window)
		enclosing := findEnclosingFunction(index+1, functions)
		handlerName := ""
		if enclosing != nil {
			handlerName = enclosing.Name
		} else {
			handlerName = extractRegisteredHandlerName(window, functions)
		}

		methods := extractMethods(window)
		if len(methods) == 0 && handlerName != "" {
			if method := inferMethodFromName(handlerName); method != "" {
				methods = []string{method}
			}
		}
		if len(methods) == 0 && enclosing != nil && contextScore >= 2 {
			methods = []string{"GET"}
		}
		if len(methods) == 0 {
			continue
		}

		for _, pathValue := range paths {
			if !isLikelyEndpointPath(pathValue) {
				continue
			}

			for _, method := range methods {
				confidence := contextScore
				if handlerName != "" {
					confidence++
				}
				if enclosing != nil {
					confidence++
				}
				if confidence < 3 {
					continue
				}

				endpoint := EndpointDoc{
					Name:        deriveEndpointName(handlerName, method, pathValue),
					Description: nearestComment(comments, index+1).Description,
					Method:      method,
					Path:        pathValue,
					File:        path,
					Line:        index + 1,
					Source:      "heuristic",
					Responses: []ResponseDoc{
						{
							Status:      defaultStatus(method),
							Type:        inferResponseType(enclosing),
							Description: "Auto-inferred response",
						},
					},
				}
				if enclosing != nil {
					endpoint.Signature = enclosing.Signature
					endpoint.ReturnType = enclosing.ReturnType
					endpoint.Params = enclosing.Params
					if endpoint.Description == "" {
						endpoint.Description = enclosing.Description
					}
					if endpoint.Name == "" {
						endpoint.Name = enclosing.Name
					}
				}

				updateEndpointCandidate(candidates, endpoint, confidence)
			}
		}
	}

	var endpoints []EndpointDoc
	for _, candidate := range candidates {
		endpoints = append(endpoints, candidate.EndpointDoc)
	}

	for _, fn := range functions {
		if functionHasEndpointCandidate(candidates, fn.Name) {
			continue
		}
		method := inferMethodFromName(fn.Name)
		path := inferPathFromName(fn.Name)
		if method == "" || path == "" {
			continue
		}
		key := strings.Join([]string{method, path, fn.Name, filepath.Base(fn.File)}, "|")
		if _, ok := candidates[key]; ok {
			continue
		}
		endpoints = append(endpoints, EndpointDoc{
			Name:        fn.Name,
			Description: fn.Description,
			Method:      method,
			Path:        path,
			Signature:   fn.Signature,
			ReturnType:  fn.ReturnType,
			File:        fn.File,
			Line:        fn.Line,
			Params:      fn.Params,
			Source:      "heuristic",
			Responses: []ResponseDoc{
				{
					Status:      defaultStatus(method),
					Type:        inferResponseType(&fn),
					Description: "Auto-inferred from function naming",
				},
			},
		})
	}

	slices.SortFunc(endpoints, func(a, b EndpointDoc) int {
		if compare := strings.Compare(a.Path, b.Path); compare != 0 {
			return compare
		}
		if compare := strings.Compare(a.Method, b.Method); compare != 0 {
			return compare
		}
		return a.Line - b.Line
	})
	return endpoints
}

func attachHeuristicRoutes(functions []FunctionDoc, endpoints []EndpointDoc) {
	for index := range functions {
		if functions[index].Route != nil {
			continue
		}
		matches := make([]EndpointDoc, 0, 1)
		for _, endpoint := range endpoints {
			if endpoint.Name != functions[index].Name {
				continue
			}
			matches = append(matches, endpoint)
		}
		if len(matches) != 1 {
			continue
		}
		functions[index].Route = &RouteDoc{
			Method: matches[0].Method,
			Path:   matches[0].Path,
		}
		if functions[index].Description == "" {
			functions[index].Description = matches[0].Description
		}
		if len(functions[index].Responses) == 0 {
			functions[index].Responses = matches[0].Responses
		}
	}
}

func inferH2ORegistrationEndpoints(path string, lines []string, comments []commentBlock, functions []FunctionDoc) []endpointCandidate {
	candidates := make([]endpointCandidate, 0)
	for index, line := range lines {
		if matches := h2oRegisterHandlerPattern.FindStringSubmatch(line); len(matches) == 3 {
			pathValue, err := strconv.Unquote(`"` + matches[1] + `"`)
			if err != nil {
				pathValue = matches[1]
			}
			handlerName := matches[2]
			fn := findFunctionByName(functions, handlerName)
			methods := inferMethodsForFunction(lines, fn, line)
			description := nearestComment(comments, index+1).Description
			for _, method := range methods {
				candidates = append(candidates, endpointCandidate{
					EndpointDoc: buildEndpointDoc(path, index+1, pathValue, method, handlerName, description, fn, "Extracted from H2O register_handler call"),
					Confidence:  8,
				})
			}
		}

		if matches := h2oConfigRegisterPathPattern.FindStringSubmatch(line); len(matches) == 2 {
			pathValue, err := strconv.Unquote(`"` + matches[1] + `"`)
			if err != nil {
				pathValue = matches[1]
			}
			description := nearestComment(comments, index+1).Description
			method := inferH2OPathMethod(lines, index)
			name := "h2o path " + pathValue
			candidates = append(candidates, endpointCandidate{
				EndpointDoc: buildEndpointDoc(path, index+1, pathValue, method, name, description, nil, "Extracted from h2o_config_register_path"),
				Confidence:  7,
			})
		}
	}
	return candidates
}

func isH2ORegistrationLine(line string) bool {
	return h2oRegisterHandlerPattern.MatchString(line) || h2oConfigRegisterPathPattern.MatchString(line)
}

func inferH2OPathMethod(lines []string, index int) string {
	window := buildWindow(lines, index, 0, 3)
	switch {
	case strings.Contains(window, "h2o_file_register") || strings.Contains(window, "redirect"):
		return "GET"
	default:
		return "ANY"
	}
}

func inferMethodsForFunction(lines []string, fn *FunctionDoc, inline string) []string {
	methods := extractMethods(inline)
	if len(methods) > 0 {
		return methods
	}
	if fn != nil {
		body := functionBody(lines, fn)
		methods = extractMethods(body)
		if len(methods) > 0 {
			return methods
		}
		methods = extractMethods(fn.Description)
		if len(methods) > 0 {
			return methods
		}
		if method := inferMethodFromName(fn.Name); method != "" {
			return []string{method}
		}
	}
	return []string{"ANY"}
}

func functionBody(lines []string, fn *FunctionDoc) string {
	if fn == nil || fn.Line <= 0 || fn.EndLine <= 0 || fn.Line > len(lines) {
		return ""
	}
	end := fn.EndLine
	if end > len(lines) {
		end = len(lines)
	}
	if end < fn.Line {
		return ""
	}
	return strings.Join(lines[fn.Line-1:end], "\n")
}

func findFunctionByName(functions []FunctionDoc, name string) *FunctionDoc {
	for index := range functions {
		if functions[index].Name == name {
			return &functions[index]
		}
	}
	return nil
}

func buildEndpointDoc(file string, line int, pathValue string, method string, name string, description string, fn *FunctionDoc, fallbackDescription string) EndpointDoc {
	endpoint := EndpointDoc{
		Name:        deriveEndpointName(name, method, pathValue),
		Description: fallbackDescriptionText(description, fallbackDescription),
		Method:      method,
		Path:        pathValue,
		File:        file,
		Line:        line,
		Source:      "heuristic",
		Responses: []ResponseDoc{
			{
				Status:      defaultStatus(method),
				Type:        inferResponseType(fn),
				Description: "Auto-inferred response",
			},
		},
	}
	if fn != nil {
		endpoint.Signature = fn.Signature
		endpoint.ReturnType = fn.ReturnType
		endpoint.Params = fn.Params
		if endpoint.Description == "" {
			endpoint.Description = fallbackDescriptionText(fn.Description, fallbackDescription)
		}
		if endpoint.Name == "" {
			endpoint.Name = fn.Name
		}
	}
	return endpoint
}

func addEndpointCandidates(store map[string]endpointCandidate, additions []endpointCandidate) {
	for _, candidate := range additions {
		updateEndpointCandidate(store, candidate.EndpointDoc, candidate.Confidence)
	}
}

func updateEndpointCandidate(store map[string]endpointCandidate, endpoint EndpointDoc, confidence int) {
	key := strings.Join([]string{endpoint.Method, endpoint.Path, endpoint.Name, filepath.Base(endpoint.File)}, "|")
	if current, ok := store[key]; !ok || confidence > current.Confidence {
		store[key] = endpointCandidate{
			EndpointDoc: endpoint,
			Confidence:  confidence,
		}
	}
}

func functionHasEndpointCandidate(store map[string]endpointCandidate, functionName string) bool {
	for _, candidate := range store {
		if candidate.Name == functionName {
			return true
		}
	}
	return false
}

func buildWindow(lines []string, index int, before int, after int) string {
	start := index - before
	if start < 0 {
		start = 0
	}
	end := index + after + 1
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

func extractPaths(text string) []string {
	matches := cStringLiteralPattern.FindAllString(text, -1)
	paths := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		value, err := strconv.Unquote(match)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(value, "/") || seen[value] {
			continue
		}
		seen[value] = true
		paths = append(paths, value)
	}
	return paths
}

func extractMethods(text string) []string {
	upper := strings.ToUpper(text)
	matches := httpMethodPattern.FindAllStringSubmatch(upper, -1)
	methods := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		method := match[1]
		if seen[method] {
			continue
		}
		seen[method] = true
		methods = append(methods, method)
	}
	return methods
}

func routeContextScore(text string) int {
	upper := strings.ToUpper(text)
	score := 0
	if len(extractPaths(text)) > 0 {
		score++
	}
	if len(extractMethods(text)) > 0 {
		score += 2
	}
	keywords := []string{
		"URI", "URL", "ROUTE", "ENDPOINT", "REQUEST", "METHOD", "MG_MATCH",
		"MG_HTTP_MATCH_URI", "ADD_ENDPOINT", "STRCMP", "STRNCMP", "HTTP",
	}
	for _, keyword := range keywords {
		if strings.Contains(upper, keyword) {
			score++
			break
		}
	}
	return score
}

func findEnclosingFunction(line int, functions []FunctionDoc) *FunctionDoc {
	for index := range functions {
		if functions[index].Line <= line && functions[index].EndLine >= line {
			return &functions[index]
		}
	}
	return nil
}

func extractRegisteredHandlerName(window string, functions []FunctionDoc) string {
	for _, fn := range functions {
		if fn.Name == "" {
			continue
		}
		if strings.Contains(window, "&"+fn.Name) {
			return fn.Name
		}
		if strings.Contains(window, "callback") && containsIdentifier(window, fn.Name) {
			return fn.Name
		}
	}
	return ""
}

func containsIdentifier(text string, identifier string) bool {
	index := strings.Index(text, identifier)
	for index >= 0 {
		beforeOK := index == 0 || !isIdentifierChar(rune(text[index-1]))
		afterIndex := index + len(identifier)
		afterOK := afterIndex >= len(text) || !isIdentifierChar(rune(text[afterIndex]))
		if beforeOK && afterOK {
			return true
		}
		nextStart := index + len(identifier)
		if nextStart >= len(text) {
			break
		}
		offset := strings.Index(text[nextStart:], identifier)
		if offset == -1 {
			break
		}
		index = nextStart + offset
	}
	return false
}

func isIdentifierChar(value rune) bool {
	return value == '_' || (value >= 'a' && value <= 'z') || (value >= 'A' && value <= 'Z') || (value >= '0' && value <= '9')
}

func inferMethodFromName(name string) string {
	tokens := tokenizeIdentifier(name)
	if len(tokens) == 0 {
		return ""
	}
	switch tokens[0] {
	case "get", "list", "fetch", "read":
		return "GET"
	case "create", "add", "post":
		return "POST"
	case "update", "set", "put":
		return "PUT"
	case "delete", "remove", "del":
		return "DELETE"
	case "patch":
		return "PATCH"
	case "any":
		return "ANY"
	default:
		return ""
	}
}

func inferPathFromName(name string) string {
	tokens := tokenizeIdentifier(name)
	if len(tokens) < 2 {
		return ""
	}
	verbs := map[string]bool{
		"get": true, "list": true, "fetch": true, "read": true,
		"create": true, "add": true, "post": true,
		"update": true, "set": true, "put": true,
		"delete": true, "remove": true, "del": true, "patch": true,
	}
	filtered := make([]string, 0, len(tokens))
	for index, token := range tokens {
		if index == 0 && verbs[token] {
			continue
		}
		if token == "by" || token == "handler" || token == "callback" || token == "api" {
			continue
		}
		filtered = append(filtered, token)
	}
	if len(filtered) == 0 {
		return ""
	}
	path := "/" + strings.Join(filtered, "-")
	if !strings.HasSuffix(path, "s") && len(filtered) == 1 {
		path += "s"
	}
	return path
}

func tokenizeIdentifier(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	name = strings.ReplaceAll(name, "-", "_")
	parts := strings.Split(strings.ToLower(name), "_")
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		tokens = append(tokens, part)
	}
	return tokens
}

func isLikelyEndpointPath(path string) bool {
	if path == "" || path == "/" || strings.Contains(path, " ") {
		return false
	}
	lower := strings.ToLower(path)
	for _, suffix := range []string{".html", ".css", ".js", ".png", ".jpg", ".jpeg", ".svg", ".ico"} {
		if strings.HasSuffix(lower, suffix) {
			return false
		}
	}
	return strings.ContainsAny(lower, "abcdefghijklmnopqrstuvwxyz")
}

func inferResponseType(fn *FunctionDoc) string {
	if fn == nil || fn.ReturnType == "" {
		return "json"
	}
	return fn.ReturnType
}

func fallbackDescriptionText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func deriveEndpointName(handlerName string, method string, path string) string {
	if handlerName != "" {
		return handlerName
	}
	if method != "" && path != "" {
		return strings.ToLower(method) + " " + path
	}
	return ""
}
