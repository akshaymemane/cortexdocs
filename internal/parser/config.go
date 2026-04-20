package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func discoverConfigFiles(root string) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, fmt.Errorf("path does not exist: %s: %w", root, err)
	}

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "output" || name == "web" || name == "node_modules" || name == "deps" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		base := strings.ToLower(filepath.Base(path))
		if ext == ".yaml" || ext == ".yml" || ext == ".conf" || base == "h2o.conf" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk config files: %w", err)
	}

	slices.Sort(files)
	return files, nil
}

func parseConfigFile(path string) ([]EndpointDoc, error) {
	lines, err := readSourceLines(path)
	if err != nil {
		return nil, fmt.Errorf("read config from %s: %w", path, err)
	}
	if !looksLikeH2OConfig(lines) {
		return nil, nil
	}

	var endpoints []EndpointDoc
	inPaths := false
	pathsIndent := -1

	for index := 0; index < len(lines); index++ {
		raw := lines[index]
		trimmed := stripInlineComment(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}

		indent := leadingIndent(raw)
		if !inPaths {
			if trimmed == "paths:" {
				inPaths = true
				pathsIndent = indent
			}
			continue
		}

		if indent <= pathsIndent && !strings.HasPrefix(trimmed, "-") {
			inPaths = false
			if trimmed == "paths:" {
				inPaths = true
				pathsIndent = indent
			}
			continue
		}

		pathValue, ok := parseConfigPathKey(trimmed)
		if !ok {
			continue
		}

		description, method := describeH2OPathBlock(lines, index, indent)
		endpoints = append(endpoints, EndpointDoc{
			Name:        "h2o path " + pathValue,
			Description: description,
			Method:      method,
			Path:        pathValue,
			File:        path,
			Line:        index + 1,
			Source:      "config",
			ReturnType:  "configured-route",
			Responses: []ResponseDoc{
				{
					Status:      defaultStatus(method),
					Type:        "configured-route",
					Description: "Extracted from H2O config",
				},
			},
		})
	}

	return endpoints, nil
}

func looksLikeH2OConfig(lines []string) bool {
	hasPaths := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "paths:" {
			hasPaths = true
		}
		if strings.HasPrefix(trimmed, "hosts:") || strings.Contains(trimmed, "file.dir:") || strings.Contains(trimmed, "fastcgi.connect:") || strings.Contains(trimmed, "proxy.reverse.url:") {
			return hasPaths || true
		}
	}
	return hasPaths
}

func parseConfigPathKey(trimmed string) (string, bool) {
	if !strings.HasSuffix(trimmed, ":") {
		return "", false
	}
	key := strings.TrimSuffix(trimmed, ":")
	key = strings.TrimSpace(key)
	if len(key) >= 2 && ((key[0] == '"' && key[len(key)-1] == '"') || (key[0] == '\'' && key[len(key)-1] == '\'')) {
		key = key[1 : len(key)-1]
	}
	if !strings.HasPrefix(key, "/") {
		return "", false
	}
	if !isLikelyEndpointPath(key) {
		return "", false
	}
	return key, true
}

func describeH2OPathBlock(lines []string, start int, parentIndent int) (string, string) {
	description := "Configured route from H2O YAML"
	method := "ANY"

	for index := start + 1; index < len(lines); index++ {
		raw := lines[index]
		trimmed := stripInlineComment(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		indent := leadingIndent(raw)
		if indent <= parentIndent {
			break
		}

		switch {
		case strings.Contains(trimmed, "file.dir:") || strings.Contains(trimmed, "file.file:"):
			description = "Static file route from H2O config"
			method = "GET"
		case strings.Contains(trimmed, "redirect:"):
			description = "Redirect route from H2O config"
			method = "GET"
		case strings.Contains(trimmed, "fastcgi.connect:"):
			description = "FastCGI route from H2O config"
		case strings.Contains(trimmed, "proxy.reverse.url:") || strings.Contains(trimmed, "proxy.preserve-host:") || strings.Contains(trimmed, "proxy.timeout.io:"):
			description = "Proxy route from H2O config"
		case strings.Contains(trimmed, "mruby.handler:"):
			description = "mruby route from H2O config"
		case strings.Contains(trimmed, "status:"):
			description = "Status route from H2O config"
			method = "GET"
		}
	}

	return description, method
}

func stripInlineComment(line string) string {
	if index := strings.Index(line, " #"); index >= 0 {
		return strings.TrimSpace(line[:index])
	}
	return line
}

func leadingIndent(line string) int {
	count := 0
	for _, char := range line {
		if char == ' ' {
			count++
			continue
		}
		if char == '\t' {
			count += 2
			continue
		}
		break
	}
	return count
}
