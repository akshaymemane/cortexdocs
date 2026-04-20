package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/akshaymemane/cortexdocs/internal/generator"
	"github.com/akshaymemane/cortexdocs/internal/parser"
	"github.com/akshaymemane/cortexdocs/internal/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve working dir: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		fs := flag.NewFlagSet("generate", flag.ExitOnError)
		nameFlag := fs.String("name", "", "API name (default: CortexDocs)")
		outputFlag := fs.String("output", filepath.Join(root, "output", "api.json"), "Output JSON path")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "missing source path")
			usage()
			os.Exit(1)
		}
		source := fs.Arg(0)
		if err := runGenerate(source, *outputFlag, *nameFlag); err != nil {
			fmt.Fprintf(os.Stderr, "generate failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s from %s\n", *outputFlag, source)

	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		portFlag := fs.String("port", "8080", "Listen port")
		_ = fs.Parse(os.Args[2:])
		addr := ":" + *portFlag
		if err := server.Start(addr, root); err != nil {
			fmt.Fprintf(os.Stderr, "serve failed: %v\n", err)
			os.Exit(1)
		}

	default:
		usage()
		os.Exit(1)
	}
}

func runGenerate(source, output, name string) error {
	parsed, err := parser.ParsePath(source)
	if err != nil {
		return err
	}

	spec := generator.BuildSpec(source, name, parsed)
	if err := generator.WriteJSON(spec, output); err != nil {
		return err
	}
	return nil
}

func usage() {
	fmt.Println(`CortexDocs

Usage:
  go run ./cli generate [--name "My API"] [--output ./output/api.json] <path>
  go run ./cli serve [--port 8080]`)
}
