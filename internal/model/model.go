package model

type Spec struct {
	Name        string     `json:"name"`
	GeneratedAt string     `json:"generatedAt"`
	SourcePath  string     `json:"sourcePath"`
	Files       []string   `json:"files"`
	Summary     Summary    `json:"summary"`
	Endpoints   []Endpoint `json:"endpoints"`
	Functions   []Function `json:"functions"`
	Structs     []Struct   `json:"structs"`
	Enums       []Enum     `json:"enums"`
	Warnings    []string   `json:"warnings,omitempty"`
}

type Summary struct {
	EndpointCount int `json:"endpointCount"`
	FunctionCount int `json:"functionCount"`
	StructCount   int `json:"structCount"`
	EnumCount     int `json:"enumCount"`
}

type Endpoint struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Description string      `json:"description"`
	Signature   string      `json:"signature"`
	ReturnType  string      `json:"returnType"`
	File        string      `json:"file"`
	Line        int         `json:"line"`
	Params      []Parameter `json:"params"`
	Responses   []Response  `json:"responses"`
	Source      string      `json:"source,omitempty"`
	Deprecated  bool        `json:"deprecated,omitempty"`
	Example     string      `json:"example,omitempty"`
}

type Function struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Signature   string      `json:"signature"`
	ReturnType  string      `json:"returnType"`
	File        string      `json:"file"`
	Line        int         `json:"line"`
	Params      []Parameter `json:"params"`
	Deprecated  bool        `json:"deprecated,omitempty"`
	Example     string      `json:"example,omitempty"`
}

type Parameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Direction   string `json:"direction,omitempty"`
}

type Response struct {
	Status      string `json:"status"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type Struct struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Fields      []Field `json:"fields"`
}

type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type Enum struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	File        string      `json:"file"`
	Line        int         `json:"line"`
	Values      []EnumValue `json:"values"`
}

type EnumValue struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}
