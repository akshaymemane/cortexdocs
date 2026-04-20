package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func ParsePath(root string) (ParseResult, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ParseResult{}, fmt.Errorf("resolve path: %w", err)
	}

	cFiles, err := discoverCFiles(absRoot)
	if err != nil {
		return ParseResult{}, err
	}
	configFiles, err := discoverConfigFiles(absRoot)
	if err != nil {
		return ParseResult{}, err
	}
	if len(cFiles) == 0 && len(configFiles) == 0 {
		return ParseResult{}, fmt.Errorf("no supported source files found in %s", absRoot)
	}

	result := ParseResult{
		Files: append(append([]string{}, cFiles...), configFiles...),
	}

	for _, file := range cFiles {
		fileResult, err := parseFile(file)
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		result.Endpoints = append(result.Endpoints, fileResult.Endpoints...)
		result.Functions = append(result.Functions, fileResult.Functions...)
		result.Structs = append(result.Structs, fileResult.Structs...)
		result.Enums = append(result.Enums, fileResult.Enums...)
	}

	for _, file := range configFiles {
		endpoints, err := parseConfigFile(file)
		if err != nil {
			result.Warnings = append(result.Warnings, err.Error())
			continue
		}
		result.Endpoints = append(result.Endpoints, endpoints...)
	}

	return result, nil
}

func discoverCFiles(root string) ([]string, error) {
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
			if name == ".git" || name == "output" || name == "web" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".c" || ext == ".h" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk files: %w", err)
	}

	slices.Sort(files)
	return files, nil
}

func parseFile(path string) (ParseResult, error) {
	comments, err := parseCommentBlocks(path)
	if err != nil {
		return ParseResult{}, fmt.Errorf("read comments from %s: %w", path, err)
	}
	lines, err := readSourceLines(path)
	if err != nil {
		return ParseResult{}, fmt.Errorf("read source from %s: %w", path, err)
	}

	cmd := exec.Command(
		"clang",
		"-Xclang", "-ast-dump=json",
		"-fsyntax-only",
		"-fparse-all-comments",
		"-x", "c",
		path,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return ParseResult{}, fmt.Errorf("clang parse failed for %s: %s", path, message)
	}

	var root astNode
	if err := json.Unmarshal(stdout.Bytes(), &root); err != nil {
		return ParseResult{}, fmt.Errorf("decode clang AST for %s: %w", path, err)
	}

	result := ParseResult{}
	for _, node := range root.Inner {
		walkNode(path, comments, node, &result)
	}
	result.Endpoints = inferEndpoints(path, lines, comments, result.Functions)
	attachHeuristicRoutes(result.Functions, result.Endpoints)

	return result, nil
}

func walkNode(sourceFile string, comments []commentBlock, node *astNode, result *ParseResult) {
	if node == nil {
		return
	}

	if node.IsImplicit {
		return
	}

	switch node.Kind {
	case "FunctionDecl":
		if functionDoc, ok := buildFunctionDoc(sourceFile, comments, node); ok {
			result.Functions = append(result.Functions, functionDoc)
		}
	case "RecordDecl":
		if structDoc, ok := buildStructDoc(sourceFile, comments, node); ok {
			result.Structs = append(result.Structs, structDoc)
		}
	case "EnumDecl":
		if enumDoc, ok := buildEnumDoc(sourceFile, comments, node); ok {
			result.Enums = append(result.Enums, enumDoc)
		}
	}

	for _, child := range node.Inner {
		walkNode(sourceFile, comments, child, result)
	}
}

func buildFunctionDoc(sourceFile string, comments []commentBlock, node *astNode) (FunctionDoc, bool) {
	if node.Name == "" || !sameSourceFile(sourceFile, node.Loc) {
		return FunctionDoc{}, false
	}

	comment := nearestComment(comments, node.Loc.Line)
	params := make([]ParamDoc, 0, len(node.Inner))
	for _, child := range node.Inner {
		switch child.Kind {
		case "ParmVarDecl":
			param := ParamDoc{
				Name: child.Name,
				Type: safeQualType(child.Type),
			}
			params = append(params, param)
		}
	}

	for index := range params {
		if commentParam, ok := comment.Params[params[index].Name]; ok {
			if commentParam.Description != "" {
				params[index].Description = commentParam.Description
			}
			if commentParam.Direction != "" {
				params[index].Direction = commentParam.Direction
			}
		}
	}

	returnType := extractReturnType(safeQualType(node.Type))
	responses := comment.Responses
	if len(responses) == 0 && comment.Route != nil {
		responses = []ResponseDoc{
			{
				Status:      defaultStatus(comment.Route.Method),
				Type:        returnType,
				Description: "Auto-inferred response",
			},
		}
	}

	routeSource := ""
	if comment.Route != nil {
		routeSource = "docblock"
	}
	return FunctionDoc{
		Name:        node.Name,
		Signature:   safeQualType(node.Type),
		ReturnType:  returnType,
		File:        sourceFile,
		Line:        node.Loc.Line,
		EndLine:     node.Range.End.Line,
		Params:      params,
		Description: comment.Description,
		Route:       comment.Route,
		RouteSource: routeSource,
		Responses:   responses,
		Deprecated:  comment.Deprecated,
		Example:     comment.Example,
	}, true
}

func buildStructDoc(sourceFile string, comments []commentBlock, node *astNode) (StructDoc, bool) {
	if node.Name == "" || node.TagUsed != "struct" || !node.CompleteDefinition || !sameSourceFile(sourceFile, node.Loc) {
		return StructDoc{}, false
	}

	fields := make([]FieldDoc, 0, len(node.Inner))
	for _, child := range node.Inner {
		if child.Kind != "FieldDecl" {
			continue
		}
		fields = append(fields, FieldDoc{
			Name: child.Name,
			Type: safeQualType(child.Type),
		})
	}

	return StructDoc{
		Name:        node.Name,
		Description: nearestComment(comments, node.Loc.Line).Description,
		File:        sourceFile,
		Line:        node.Loc.Line,
		Fields:      fields,
	}, true
}

func buildEnumDoc(sourceFile string, comments []commentBlock, node *astNode) (EnumDoc, bool) {
	if node.Name == "" || !sameSourceFile(sourceFile, node.Loc) {
		return EnumDoc{}, false
	}

	values := make([]EnumValueDoc, 0, len(node.Inner))
	for _, child := range node.Inner {
		if child.Kind != "EnumConstantDecl" {
			continue
		}
		values = append(values, EnumValueDoc{Name: child.Name})
	}

	return EnumDoc{
		Name:        node.Name,
		Description: nearestComment(comments, node.Loc.Line).Description,
		File:        sourceFile,
		Line:        node.Loc.Line,
		Values:      values,
	}, true
}

func safeQualType(value *astType) string {
	if value == nil {
		return "void"
	}
	if value.QualType == "" {
		return "void"
	}
	return value.QualType
}

func sameSourceFile(expected string, actual astLoc) bool {
	if actual.IncludedFrom != nil {
		return false
	}
	return actual.File == "" || filepath.Clean(actual.File) == filepath.Clean(expected)
}

func extractReturnType(signature string) string {
	index := strings.Index(signature, "(")
	if index == -1 {
		return signature
	}
	return strings.TrimSpace(signature[:index])
}

func defaultStatus(method string) string {
	switch strings.ToUpper(method) {
	case "POST":
		return "201"
	case "DELETE":
		return "204"
	default:
		return "200"
	}
}
