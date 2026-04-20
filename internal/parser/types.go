package parser

type ParseResult struct {
	Files     []string
	Endpoints []EndpointDoc
	Functions []FunctionDoc
	Structs   []StructDoc
	Enums     []EnumDoc
	Warnings  []string
}

type EndpointDoc struct {
	Name        string
	Description string
	Method      string
	Path        string
	Signature   string
	ReturnType  string
	File        string
	Line        int
	Params      []ParamDoc
	Responses   []ResponseDoc
	Source      string // "heuristic" or "config"
	Deprecated  bool
	Example     string
}

type FunctionDoc struct {
	Name        string
	Signature   string
	ReturnType  string
	File        string
	Line        int
	EndLine     int
	Params      []ParamDoc
	Description string
	Route       *RouteDoc
	RouteSource string // "docblock" or "heuristic"
	Responses   []ResponseDoc
	Deprecated  bool
	Example     string
}

type ParamDoc struct {
	Name        string
	Type        string
	Description string
	Direction   string
}

type ResponseDoc struct {
	Status      string
	Type        string
	Description string
}

type RouteDoc struct {
	Method string
	Path   string
}

type StructDoc struct {
	Name        string
	Description string
	File        string
	Line        int
	Fields      []FieldDoc
}

type FieldDoc struct {
	Name        string
	Type        string
	Description string
}

type EnumDoc struct {
	Name        string
	Description string
	File        string
	Line        int
	Values      []EnumValueDoc
}

type EnumValueDoc struct {
	Name        string
	Description string
}

type astNode struct {
	Kind               string      `json:"kind"`
	Name               string      `json:"name,omitempty"`
	Text               string      `json:"text,omitempty"`
	Param              string      `json:"param,omitempty"`
	Direction          string      `json:"direction,omitempty"`
	TagUsed            string      `json:"tagUsed,omitempty"`
	IsImplicit         bool        `json:"isImplicit,omitempty"`
	CompleteDefinition bool        `json:"completeDefinition,omitempty"`
	Type               *astType    `json:"type,omitempty"`
	Loc                astLoc      `json:"loc,omitempty"`
	Range              astRange    `json:"range,omitempty"`
	Inner              []*astNode  `json:"inner,omitempty"`
	OwnedTagDecl       *astNodeRef `json:"ownedTagDecl,omitempty"`
	Decl               *astNodeRef `json:"decl,omitempty"`
}

type astType struct {
	QualType string `json:"qualType,omitempty"`
}

type astNodeRef struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

type astLoc struct {
	File         string           `json:"file,omitempty"`
	Line         int              `json:"line,omitempty"`
	Col          int              `json:"col,omitempty"`
	Offset       int              `json:"offset,omitempty"`
	TokLen       int              `json:"tokLen,omitempty"`
	IncludedFrom *astIncludedFrom `json:"includedFrom,omitempty"`
}

type astRange struct {
	Begin astLoc `json:"begin"`
	End   astLoc `json:"end"`
}

type astIncludedFrom struct {
	File string `json:"file,omitempty"`
}

type commentDoc struct {
	Description string
	Route       *RouteDoc
	Params      map[string]ParamDoc
	Responses   []ResponseDoc
	Deprecated  bool
	Example     string
}

type commentBlock struct {
	EndLine int
	Doc     commentDoc
}
