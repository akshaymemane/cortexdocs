package parser

import (
	"bufio"
	"os"
	"strings"
)

func parseCommentBlocks(path string) ([]commentBlock, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		blocks      []commentBlock
		buffer      []string
		inBlock     bool
		singleBuf   []string
		inSingleDoc bool
		lineNumber  int
	)

	flushSingle := func(endLine int) {
		if len(singleBuf) > 0 {
			blocks = append(blocks, commentBlock{
				EndLine: endLine,
				Doc:     parseComment(singleBuf),
			})
			singleBuf = nil
		}
		inSingleDoc = false
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		isSingleDoc := strings.HasPrefix(trimmed, "///") || strings.HasPrefix(trimmed, "//!")

		// Flush pending single-line block when it ends
		if inSingleDoc && !isSingleDoc && !inBlock {
			flushSingle(lineNumber - 1)
		}

		if isSingleDoc && !inBlock {
			inSingleDoc = true
			singleBuf = append(singleBuf, strings.TrimSpace(trimmed[3:]))
			continue
		}

		if !inBlock && strings.HasPrefix(trimmed, "/**") {
			inBlock = true
			buffer = []string{trimmed}
			if strings.Contains(trimmed, "*/") {
				blocks = append(blocks, commentBlock{
					EndLine: lineNumber,
					Doc:     parseComment(buffer),
				})
				inBlock = false
				buffer = nil
			}
			continue
		}

		if inBlock {
			buffer = append(buffer, trimmed)
			if strings.Contains(trimmed, "*/") {
				blocks = append(blocks, commentBlock{
					EndLine: lineNumber,
					Doc:     parseComment(buffer),
				})
				inBlock = false
				buffer = nil
			}
		}
	}

	// Flush any trailing single-line block
	flushSingle(lineNumber)

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return blocks, nil
}

func parseComment(lines []string) commentDoc {
	comment := commentDoc{Params: map[string]ParamDoc{}}
	descriptionLines := make([]string, 0, len(lines))

	for _, line := range lines {
		normalized := normalizeCommentLine(line)
		if normalized == "" {
			continue
		}

		switch {
		case strings.HasPrefix(normalized, "@route "):
			fields := strings.Fields(strings.TrimPrefix(normalized, "@route "))
			if len(fields) >= 2 {
				comment.Route = &RouteDoc{
					Method: strings.ToUpper(fields[0]),
					Path:   fields[1],
				}
			}

		case strings.HasPrefix(normalized, "@desc "):
			descriptionLines = append(descriptionLines, strings.TrimSpace(strings.TrimPrefix(normalized, "@desc ")))

		case strings.HasPrefix(normalized, "@param "):
			rest := strings.TrimSpace(strings.TrimPrefix(normalized, "@param "))
			direction := "in"
			if strings.HasPrefix(rest, "[") {
				end := strings.Index(rest, "]")
				if end > 0 {
					direction = strings.ToLower(rest[1:end])
					rest = strings.TrimSpace(rest[end+1:])
				}
			}
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				name := fields[0]
				description := ""
				if len(fields) > 1 {
					description = strings.Join(fields[1:], " ")
				}
				comment.Params[name] = ParamDoc{
					Name:        name,
					Description: description,
					Direction:   direction,
				}
			}

		case strings.HasPrefix(normalized, "@response "):
			rest := strings.TrimSpace(strings.TrimPrefix(normalized, "@response "))
			fields := strings.Fields(rest)
			if len(fields) >= 2 {
				response := ResponseDoc{
					Status: fields[0],
					Type:   fields[1],
				}
				if len(fields) > 2 {
					response.Description = strings.Join(fields[2:], " ")
				}
				comment.Responses = append(comment.Responses, response)
			}

		case normalized == "@deprecated" || strings.HasPrefix(normalized, "@deprecated "):
			comment.Deprecated = true

		case strings.HasPrefix(normalized, "@example"):
			rest := strings.TrimSpace(strings.TrimPrefix(normalized, "@example"))
			if rest != "" {
				comment.Example = rest
			}

		case strings.HasPrefix(normalized, "@"):
			// unknown tag — skip

		default:
			descriptionLines = append(descriptionLines, normalized)
		}
	}

	comment.Description = strings.TrimSpace(strings.Join(descriptionLines, " "))
	return comment
}

func nearestComment(blocks []commentBlock, line int) commentDoc {
	bestDistance := 3
	for index := len(blocks) - 1; index >= 0; index-- {
		if blocks[index].EndLine > line {
			continue
		}
		distance := line - blocks[index].EndLine
		if distance >= 0 && distance <= bestDistance {
			return blocks[index].Doc
		}
	}
	return commentDoc{Params: map[string]ParamDoc{}}
}

func normalizeCommentLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "/**")
	line = strings.TrimSuffix(line, "*/")
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimSpace(line)
	line = strings.Join(strings.Fields(line), " ")
	return line
}
