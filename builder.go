package yourbatis

import (
	"fmt"
	"strings"
)

type Dialect string

const (
	DialectQuestion Dialect = "question"
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
	DialectMySQL    Dialect = "mysql"
)

func (dialect Dialect) Placeholder(position int) string {
	if dialect == DialectPostgres {
		return fmt.Sprintf("$%d", position)
	}
	return "?"
}

// Feature describes a database capability that changes how a statement must
// be executed. It is intentionally limited to the databases supported by this
// project.
type Feature string

const (
	FeatureDMLReturning Feature = "dml_returning"
	FeatureLastInsertID Feature = "last_insert_id"
)

func (dialect Dialect) Supports(feature Feature) bool {
	switch feature {
	case FeatureDMLReturning:
		return dialect == DialectPostgres || dialect == DialectSQLite
	case FeatureLastInsertID:
		return dialect == DialectSQLite || dialect == DialectMySQL
	default:
		return false
	}
}

func (dialect Dialect) Supported() bool {
	switch dialect {
	case DialectPostgres, DialectSQLite, DialectMySQL:
		return true
	default:
		return false
	}
}

type Argument struct {
	Name      string
	Value     any
	Sensitive bool
}

type BoundSQL struct {
	SQL  string
	Args []Argument
}

func (bound BoundSQL) Values() []any {
	values := make([]any, len(bound.Args))
	for index, argument := range bound.Args {
		values[index] = argument.Value
	}
	return values
}

type ArgOption func(*Argument)

func Sensitive() ArgOption {
	return func(argument *Argument) {
		argument.Sensitive = true
	}
}

type sqlPart struct {
	text         string
	arg          *Argument
	group        *trimmedGroup
	sqlNodeStart bool
}

type trimmedGroup struct {
	parts           []sqlPart
	prefix          string
	prefixOverrides []string
	suffix          string
	suffixOverrides []string
}

// Builder stores SQL as text/argument tokens. Placeholders are rendered only in
// Build, which lets nested trim/where/foreach fragments remain dialect-neutral.
type Builder struct {
	dialect           Dialect
	parts             []sqlPart
	hasIterations     bool
	hasForeachContent bool
}

func NewBuilder(dialect Dialect) *Builder {
	return &Builder{dialect: dialect}
}

// BeginSQLNode records one DynamicContext.appendSql call. MyBatis joins each
// appendSql value with one space, including empty values. Text and Arg calls
// following this marker remain contiguous within the same SQL node.
func (builder *Builder) BeginSQLNode() {
	if builder == nil {
		return
	}
	builder.parts = append(builder.parts, sqlPart{sqlNodeStart: true})
}

func (builder *Builder) Text(text string) {
	if builder == nil || text == "" {
		return
	}
	builder.parts = append(builder.parts, sqlPart{text: text})
}

func (builder *Builder) Arg(name string, value any, options ...ArgOption) {
	if builder == nil {
		return
	}
	argument := Argument{Name: name, Value: value}
	for _, option := range options {
		if option != nil {
			option(&argument)
		}
	}
	builder.parts = append(builder.parts, sqlPart{arg: &argument})
}

func (builder *Builder) Append(other *Builder) {
	if builder == nil || other == nil {
		return
	}
	builder.parts = append(builder.parts, cloneParts(other.parts)...)
}

// AppendSeparated appends non-empty SQL and emits separator only between
// previously appended non-empty fragments. It is the Go runtime equivalent of
// MyBatis' foreach PrefixedContext behavior.
func (builder *Builder) AppendSeparated(other *Builder, separator string) {
	if builder == nil || other == nil {
		return
	}
	builder.hasIterations = true
	if hasSQLNodeStarts(other.parts) {
		builder.appendMyBatisForeachItem(other, separator)
		return
	}
	if !other.HasContent() {
		return
	}
	if builder.HasContent() {
		builder.Text(separator)
	}
	builder.Append(other)
}

func (builder *Builder) appendMyBatisForeachItem(other *Builder, separator string) {
	parts := cloneParts(other.parts)
	contentStart := firstContentSQLNode(parts, builder.dialect)
	if contentStart < 0 {
		// PrefixedContext delegates blank SQL nodes without applying its prefix.
		builder.parts = append(builder.parts, parts...)
		return
	}
	prefix := ""
	if builder.hasForeachContent {
		prefix = separator
	}
	builder.parts = append(builder.parts, parts[:contentStart]...)
	builder.parts = append(builder.parts, sqlPart{sqlNodeStart: true})
	if prefix != "" {
		builder.parts = append(builder.parts, sqlPart{text: prefix})
	}
	builder.parts = append(builder.parts, parts[contentStart:]...)
	builder.hasForeachContent = true
}

func (builder *Builder) AppendTrimmed(
	other *Builder,
	prefix string,
	prefixOverrides []string,
	suffix string,
	suffixOverrides []string,
) {
	if builder == nil || other == nil {
		return
	}
	builder.parts = append(builder.parts, sqlPart{group: &trimmedGroup{
		parts:           cloneParts(other.parts),
		prefix:          prefix,
		prefixOverrides: append([]string(nil), prefixOverrides...),
		suffix:          suffix,
		suffixOverrides: append([]string(nil), suffixOverrides...),
	}})
}

func (builder *Builder) HasContent() bool {
	return builder != nil && partsHaveContent(builder.parts)
}

// HasIterations reports whether AppendSeparated was called for at least one
// collection item, even when every item body was empty. This matches MyBatis'
// foreach decision to emit open and close for a non-empty source collection.
func (builder *Builder) HasIterations() bool {
	return builder != nil && builder.hasIterations
}

func (builder *Builder) Build() BoundSQL {
	if builder == nil {
		return BoundSQL{}
	}
	var sqlBuilder strings.Builder
	arguments := make([]Argument, 0)
	position := 1
	sqlNodeCount := 0
	renderParts(builder.parts, builder.dialect, &position, &sqlNodeCount, &sqlBuilder, &arguments)
	return BoundSQL{
		SQL:  strings.TrimSpace(sqlBuilder.String()),
		Args: arguments,
	}
}

func renderParts(
	parts []sqlPart,
	dialect Dialect,
	position *int,
	sqlNodeCount *int,
	sqlBuilder *strings.Builder,
	arguments *[]Argument,
) {
	for _, part := range parts {
		switch {
		case part.sqlNodeStart:
			if *sqlNodeCount > 0 {
				sqlBuilder.WriteByte(' ')
			}
			*sqlNodeCount++
		case part.arg != nil:
			sqlBuilder.WriteString(dialect.Placeholder(*position))
			*position++
			*arguments = append(*arguments, *part.arg)
		case part.group != nil:
			var nested strings.Builder
			nestedArguments := make([]Argument, 0)
			nestedPosition := *position
			nestedSQLNodeCount := 0
			renderParts(
				part.group.parts,
				dialect,
				&nestedPosition,
				&nestedSQLNodeCount,
				&nested,
				&nestedArguments,
			)
			sqlText := applyTrim(
				nested.String(),
				part.group.prefix,
				part.group.prefixOverrides,
				part.group.suffix,
				part.group.suffixOverrides,
			)
			if sqlText == "" {
				continue
			}
			sqlBuilder.WriteString(sqlText)
			*position = nestedPosition
			*arguments = append(*arguments, nestedArguments...)
		default:
			sqlBuilder.WriteString(part.text)
		}
	}
}

func hasSQLNodeStarts(parts []sqlPart) bool {
	for _, part := range parts {
		if part.sqlNodeStart {
			return true
		}
	}
	return false
}

func firstContentSQLNode(parts []sqlPart, dialect Dialect) int {
	for start := 0; start < len(parts); {
		if !parts[start].sqlNodeStart {
			start++
			continue
		}
		end := start + 1
		for end < len(parts) && !parts[end].sqlNodeStart {
			end++
		}
		if sqlPartsHaveRenderedContent(parts[start+1:end], dialect) {
			return start
		}
		start = end
	}
	return -1
}

func sqlPartsHaveRenderedContent(parts []sqlPart, dialect Dialect) bool {
	var sqlBuilder strings.Builder
	arguments := make([]Argument, 0)
	position := 1
	sqlNodeCount := 0
	renderParts(parts, dialect, &position, &sqlNodeCount, &sqlBuilder, &arguments)
	return strings.TrimSpace(sqlBuilder.String()) != ""
}

func applyTrim(
	sqlText string,
	prefix string,
	prefixOverrides []string,
	suffix string,
	suffixOverrides []string,
) string {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return ""
	}
	sqlText = trimPrefixFold(sqlText, prefixOverrides)
	sqlText = trimSuffixFold(sqlText, suffixOverrides)
	if sqlText == "" {
		return ""
	}
	return prefix + sqlText + suffix
}

func trimPrefixFold(value string, overrides []string) string {
	for _, override := range overrides {
		if override == "" || len(value) < len(override) {
			continue
		}
		if strings.EqualFold(value[:len(override)], override) {
			return value[len(strings.TrimSpace(override)):]
		}
	}
	return value
}

func trimSuffixFold(value string, overrides []string) string {
	for _, override := range overrides {
		if override == "" {
			continue
		}
		trimmed := strings.TrimSpace(override)
		// Every string has an empty suffix. Ignore whitespace-only rules so
		// they cannot match vacuously and shadow later, meaningful overrides.
		if trimmed == "" {
			continue
		}
		if hasSuffixFold(value, override) || hasSuffixFold(value, trimmed) {
			return value[:len(value)-len(trimmed)]
		}
	}
	return value
}

func hasSuffixFold(value string, suffix string) bool {
	if len(value) < len(suffix) {
		return false
	}
	return strings.EqualFold(value[len(value)-len(suffix):], suffix)
}

func cloneParts(parts []sqlPart) []sqlPart {
	cloned := make([]sqlPart, len(parts))
	for index, part := range parts {
		cloned[index] = part
		if part.arg != nil {
			argument := *part.arg
			cloned[index].arg = &argument
		}
		if part.group != nil {
			group := *part.group
			group.parts = cloneParts(part.group.parts)
			group.prefixOverrides = append([]string(nil), part.group.prefixOverrides...)
			group.suffixOverrides = append([]string(nil), part.group.suffixOverrides...)
			cloned[index].group = &group
		}
	}
	return cloned
}

func partsHaveContent(parts []sqlPart) bool {
	for _, part := range parts {
		switch {
		case part.arg != nil:
			return true
		case part.group != nil && partsHaveContent(part.group.parts):
			return true
		case strings.TrimSpace(part.text) != "":
			return true
		}
	}
	return false
}
