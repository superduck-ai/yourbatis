package ir

type Span struct {
	File   string
	Line   int
	Column int
}

type Mapper struct {
	Namespace  string
	ResultMaps map[string]*ResultMap
	Statements []*Statement
}

type ResultMap struct {
	ID     string
	Type   string
	Fields []ResultField
	Source Span
}

type ResultField struct {
	Column   string
	Property string
	ID       bool
	Source   Span
}

type Command string

const (
	Select Command = "select"
	Insert Command = "insert"
	Update Command = "update"
	Delete Command = "delete"
)

type ResultMode string

const (
	ResultInfer     ResultMode = ""
	ResultOne       ResultMode = "one"
	ResultOptional  ResultMode = "optional"
	ResultMany      ResultMode = "many"
	ResultEach      ResultMode = "each"
	ResultScalar    ResultMode = "scalar"
	ResultRows      ResultMode = "rows"
	ResultExec      ResultMode = "exec"
	ResultLastID    ResultMode = "lastid"
	ResultReturning ResultMode = "returning"
)

type Statement struct {
	ID         string
	Command    Command
	ResultMode ResultMode
	ResultMap  string
	ResultType string
	Body       []Node
	Source     Span
}

type Node interface {
	Span() Span
}

// SQLNodeBoundary represents one MyBatis DynamicContext.appendSql call. The
// following fragments belong to that call until the next boundary.
type SQLNodeBoundary struct {
	Source Span
}

func (node *SQLNodeBoundary) Span() Span { return node.Source }

type Text struct {
	Value  string
	Source Span
}

func (node *Text) Span() Span { return node.Source }

type TextExpression struct {
	Expression   string
	GoExpression string
	Kind         string
	Source       Span
}

func (node *TextExpression) Span() Span { return node.Source }

type Parameter struct {
	Path      string
	GoPath    string
	Options   map[string]string
	Sensitive bool
	Source    Span
}

func (node *Parameter) Span() Span { return node.Source }

type If struct {
	Test   string
	GoTest string
	Body   []Node
	Source Span
}

func (node *If) Span() Span { return node.Source }

type When struct {
	Test   string
	GoTest string
	Body   []Node
	Source Span
}

type Choose struct {
	When      []When
	Otherwise []Node
	Source    Span
}

func (node *Choose) Span() Span { return node.Source }

type Foreach struct {
	Collection   string
	GoCollection string
	Item         string
	Index        string
	Open         string
	OpenSet      bool
	Separator    string
	Close        string
	CloseSet     bool
	Body         []Node
	Source       Span
}

func (node *Foreach) Span() Span { return node.Source }

type Trim struct {
	Prefix          string
	PrefixOverrides []string
	Suffix          string
	SuffixOverrides []string
	Body            []Node
	Source          Span
}

func (node *Trim) Span() Span { return node.Source }

type Bind struct {
	Name         string
	Expression   string
	GoExpression string
	Source       Span
}

func (node *Bind) Span() Span { return node.Source }
