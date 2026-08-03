package parsing

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type XNode struct {
	node           *Node
	name           string
	body           string
	hasBody        bool
	attributes     Properties
	attributeOrder []string
	variables      Properties
	xpathParser    *XPathParser
}

func NewXNode(xpathParser *XPathParser, node *Node, variables Properties) *XNode {
	if node == nil {
		return nil
	}

	xnode := &XNode{
		node:        node,
		name:        node.GetNodeName(),
		variables:   variables,
		xpathParser: xpathParser,
	}
	xnode.attributes, xnode.attributeOrder = xnode.parseAttributes(node)
	xnode.body, xnode.hasBody = xnode.parseBody(node)
	return xnode
}

func (xnode *XNode) NewXNode(node *Node) *XNode {
	return NewXNode(xnode.xpathParser, node, xnode.variables)
}

func (xnode *XNode) GetParent() *XNode {
	if xnode == nil || xnode.node == nil {
		return nil
	}
	parent := xnode.node.GetParentNode()
	if parent == nil || parent.GetNodeType() != ELEMENT_NODE {
		return nil
	}
	return NewXNode(xnode.xpathParser, parent, xnode.variables)
}

func (xnode *XNode) GetPath() string {
	if xnode == nil || xnode.node == nil {
		return ""
	}
	parts := make([]string, 0)
	for current := xnode.node; current != nil && current.GetNodeType() == ELEMENT_NODE; current = current.GetParentNode() {
		parts = append([]string{current.GetNodeName()}, parts...)
	}
	return strings.Join(parts, "/")
}

func (xnode *XNode) GetValueBasedIdentifier() string {
	if xnode == nil {
		return ""
	}
	current := xnode
	identifiers := make([]string, 0)
	for current != nil {
		identifier := current.GetName()
		if value, ok := current.LookupStringAttribute("id"); ok {
			identifier += "[" + strings.ReplaceAll(value, ".", "_") + "]"
		} else if value, ok := current.LookupStringAttribute("value"); ok {
			identifier += "[" + strings.ReplaceAll(value, ".", "_") + "]"
		} else if value, ok := current.LookupStringAttribute("property"); ok {
			identifier += "[" + strings.ReplaceAll(value, ".", "_") + "]"
		}
		identifiers = append([]string{identifier}, identifiers...)
		current = current.GetParent()
	}
	return strings.Join(identifiers, "_")
}

func (xnode *XNode) EvalString(expression string) string {
	if xnode == nil || xnode.xpathParser == nil {
		return ""
	}
	return xnode.xpathParser.EvalStringWithRoot(xnode.node, expression)
}

func (xnode *XNode) EvalBoolean(expression string) bool {
	if xnode == nil || xnode.xpathParser == nil {
		return false
	}
	return xnode.xpathParser.EvalBooleanWithRoot(xnode.node, expression)
}

func (xnode *XNode) EvalDouble(expression string) float64 {
	if xnode == nil || xnode.xpathParser == nil {
		return 0
	}
	return xnode.xpathParser.EvalDoubleWithRoot(xnode.node, expression)
}

func (xnode *XNode) EvalNodes(expression string) []*XNode {
	if xnode == nil || xnode.xpathParser == nil {
		return nil
	}
	return xnode.xpathParser.EvalNodesWithRoot(xnode.node, expression)
}

func (xnode *XNode) EvalNode(expression string) *XNode {
	if xnode == nil || xnode.xpathParser == nil {
		return nil
	}
	return xnode.xpathParser.EvalNodeWithRoot(xnode.node, expression)
}

func (xnode *XNode) GetNode() *Node {
	if xnode == nil {
		return nil
	}
	return xnode.node
}

func (xnode *XNode) GetName() string {
	if xnode == nil {
		return ""
	}
	return xnode.name
}

func (xnode *XNode) IsValidationEnabled() bool {
	if xnode == nil || xnode.xpathParser == nil {
		return false
	}
	return xnode.xpathParser.IsValidationEnabled()
}

func (xnode *XNode) GetPosition() Position {
	if xnode == nil || xnode.node == nil {
		return Position{}
	}
	return xnode.node.GetPosition()
}

func (xnode *XNode) GetAttributes() Properties {
	if xnode == nil || xnode.attributes == nil {
		return nil
	}
	return xnode.attributes.Clone()
}

func (xnode *XNode) LookupStringBody() (string, bool) {
	if xnode == nil || !xnode.hasBody {
		return "", false
	}
	return xnode.body, true
}

func (xnode *XNode) GetStringBody(def ...string) string {
	if value, ok := xnode.LookupStringBody(); ok {
		return value
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

func (xnode *XNode) GetBooleanBody(def ...bool) bool {
	value, ok := xnode.LookupStringBody()
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return false
	}
	return strings.EqualFold(value, "true")
}

func (xnode *XNode) GetIntBody(def ...int) int {
	value, ok := xnode.LookupStringBody()
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}
	return mustParseInt(value)
}

func (xnode *XNode) GetLongBody(def ...int64) int64 {
	value, ok := xnode.LookupStringBody()
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}
	return mustParseInt64(value)
}

func (xnode *XNode) GetDoubleBody(def ...float64) float64 {
	value, ok := xnode.LookupStringBody()
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}
	return mustParseFloat64(value)
}

func (xnode *XNode) GetFloatBody(def ...float32) float32 {
	value, ok := xnode.LookupStringBody()
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return 0
	}
	return mustParseFloat32(value)
}

func (xnode *XNode) GetEnumAttribute(enumType reflect.Type, name string, def ...any) any {
	value, ok := xnode.LookupStringAttribute(name)
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return nil
	}
	return convertStringToEnumType(value, enumType)
}

func (xnode *XNode) LookupStringAttribute(name string) (string, bool) {
	if xnode == nil || xnode.attributes == nil {
		return "", false
	}
	value, ok := xnode.attributes[name]
	return value, ok
}

func (xnode *XNode) GetStringAttributeWithSupplier(name string, defSupplier func() string) string {
	if value, ok := xnode.LookupStringAttribute(name); ok {
		return value
	}
	if defSupplier == nil {
		panic(fmt.Errorf("parsing: nil default supplier for missing attribute %q", name))
	}
	return defSupplier()
}

func (xnode *XNode) GetStringAttribute(name string, def ...string) string {
	if value, ok := xnode.LookupStringAttribute(name); ok {
		return value
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

func (xnode *XNode) LookupBooleanAttribute(name string) (bool, bool) {
	value, ok := xnode.LookupStringAttribute(name)
	if !ok {
		return false, false
	}
	return strings.EqualFold(value, "true"), true
}

func (xnode *XNode) GetBooleanAttribute(name string, def ...bool) bool {
	if value, ok := xnode.LookupBooleanAttribute(name); ok {
		return value
	}
	if len(def) > 0 {
		return def[0]
	}
	return false
}

func (xnode *XNode) LookupIntAttribute(name string) (int, bool) {
	value, ok := xnode.LookupStringAttribute(name)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (xnode *XNode) GetIntAttribute(name string, def ...int) int {
	value, ok := xnode.LookupStringAttribute(name)
	if ok {
		return mustParseInt(value)
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

func (xnode *XNode) LookupLongAttribute(name string) (int64, bool) {
	value, ok := xnode.LookupStringAttribute(name)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (xnode *XNode) GetLongAttribute(name string, def ...int64) int64 {
	value, ok := xnode.LookupStringAttribute(name)
	if ok {
		return mustParseInt64(value)
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

func (xnode *XNode) LookupDoubleAttribute(name string) (float64, bool) {
	value, ok := xnode.LookupStringAttribute(name)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(TrimJavaWhitespace(value), 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func (xnode *XNode) GetDoubleAttribute(name string, def ...float64) float64 {
	value, ok := xnode.LookupStringAttribute(name)
	if ok {
		return mustParseFloat64(value)
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

func (xnode *XNode) LookupFloatAttribute(name string) (float32, bool) {
	value, ok := xnode.LookupStringAttribute(name)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(TrimJavaWhitespace(value), 32)
	if err != nil {
		return 0, false
	}
	return float32(parsed), true
}

func (xnode *XNode) GetFloatAttribute(name string, def ...float32) float32 {
	value, ok := xnode.LookupStringAttribute(name)
	if ok {
		return mustParseFloat32(value)
	}
	if len(def) > 0 {
		return def[0]
	}
	return 0
}

func (xnode *XNode) GetChildren() []*XNode {
	if xnode == nil || xnode.node == nil {
		return nil
	}
	children := make([]*XNode, 0)
	for _, child := range xnode.node.GetChildNodes() {
		if child.GetNodeType() == ELEMENT_NODE {
			children = append(children, NewXNode(xnode.xpathParser, child, xnode.variables))
		}
	}
	return children
}

func (xnode *XNode) GetChildrenAsProperties() Properties {
	properties := NewProperties()
	for _, child := range xnode.GetChildren() {
		name, hasName := child.LookupStringAttribute("name")
		value, hasValue := child.LookupStringAttribute("value")
		if hasName && hasValue {
			properties[name] = value
		}
	}
	return properties
}

func (xnode *XNode) String() string {
	if xnode == nil {
		return ""
	}
	var builder strings.Builder
	xnode.buildToString(&builder, 0)
	return builder.String()
}

func (xnode *XNode) GetSourceXML() string {
	if xnode == nil || xnode.node == nil {
		return ""
	}

	var builder strings.Builder
	xnode.buildSourceXML(&builder, 0)
	return builder.String()
}

func (xnode *XNode) buildSourceXML(builder *strings.Builder, indentLevel int) {
	if xnode == nil || xnode.node == nil || builder == nil {
		return
	}

	switch xnode.node.GetNodeType() {
	case TEXT_NODE:
		builder.WriteString(xnode.GetStringBody(""))
	case CDATA_SECTION_NODE:
		builder.WriteString("<![CDATA[")
		builder.WriteString(xnode.GetStringBody(""))
		builder.WriteString("]]>")
	case ENTITY_REFERENCE_NODE:
		builder.WriteString("&")
		builder.WriteString(xnode.GetName())
		builder.WriteString(";")
	case ELEMENT_NODE:
		xnode.buildElementSourceXML(builder, indentLevel)
	default:
		builder.WriteString(xnode.String())
	}
}

func (xnode *XNode) buildElementSourceXML(builder *strings.Builder, indentLevel int) {
	xnode.indent(builder, indentLevel)
	builder.WriteString("<")
	builder.WriteString(xnode.name)
	for _, attributeName := range xnode.attributeOrder {
		builder.WriteString(" ")
		builder.WriteString(attributeName)
		builder.WriteString("=\"")
		builder.WriteString(xnode.attributes[attributeName])
		builder.WriteString("\"")
	}

	nodeList := xnode.node.GetChildNodes()
	if len(nodeList) == 0 {
		builder.WriteString(" />\n")
		return
	}

	builder.WriteString(">\n")
	for _, childNode := range nodeList {
		child := xnode.NewXNode(childNode)
		switch childNode.GetNodeType() {
		case ELEMENT_NODE:
			child.buildElementSourceXML(builder, indentLevel+1)
		case TEXT_NODE:
			text := TrimJavaWhitespace(child.GetStringBody(""))
			if text != "" {
				xnode.indent(builder, indentLevel+1)
				builder.WriteString(text)
				builder.WriteString("\n")
			}
		case CDATA_SECTION_NODE:
			xnode.indent(builder, indentLevel+1)
			child.buildSourceXML(builder, 0)
			builder.WriteString("\n")
		case ENTITY_REFERENCE_NODE:
			xnode.indent(builder, indentLevel+1)
			child.buildSourceXML(builder, 0)
			builder.WriteString("\n")
		}
	}
	xnode.indent(builder, indentLevel)
	builder.WriteString("</")
	builder.WriteString(xnode.name)
	builder.WriteString(">\n")
}

func (xnode *XNode) buildToString(builder *strings.Builder, indentLevel int) {
	xnode.indent(builder, indentLevel)
	builder.WriteString("<")
	builder.WriteString(xnode.name)
	for _, attributeName := range xnode.attributeOrder {
		builder.WriteString(" ")
		builder.WriteString(attributeName)
		builder.WriteString("=\"")
		builder.WriteString(xnode.attributes[attributeName])
		builder.WriteString("\"")
	}

	nodeList := xnode.node.GetChildNodes()
	if len(nodeList) == 0 {
		builder.WriteString(" />\n")
		return
	}

	builder.WriteString(">\n")
	for _, node := range nodeList {
		if node.GetNodeType() == ELEMENT_NODE {
			NewXNode(xnode.xpathParser, node, xnode.variables).buildToString(builder, indentLevel+1)
			continue
		}
		if node.GetNodeType() == ENTITY_REFERENCE_NODE {
			xnode.indent(builder, indentLevel+1)
			builder.WriteString("&")
			builder.WriteString(node.GetNodeName())
			builder.WriteString(";\n")
			continue
		}
		text, ok := xnode.getBodyData(node)
		if !ok {
			continue
		}
		text = TrimJavaWhitespace(text)
		if text != "" {
			xnode.indent(builder, indentLevel+1)
			builder.WriteString(text)
			builder.WriteString("\n")
		}
	}
	xnode.indent(builder, indentLevel)
	builder.WriteString("</")
	builder.WriteString(xnode.name)
	builder.WriteString(">\n")
}

func (xnode *XNode) indent(builder *strings.Builder, level int) {
	for i := 0; i < level; i++ {
		builder.WriteString("  ")
	}
}

func (xnode *XNode) parseAttributes(node *Node) (Properties, []string) {
	attributes := NewProperties()
	order := make([]string, 0, len(node.GetAttributes()))
	for _, attribute := range node.GetAttributes() {
		value := PropertyParser.Parse(attribute.GetNodeValue(), xnode.variables)
		attributes[attribute.GetNodeName()] = value
		order = append(order, attribute.GetNodeName())
	}
	return attributes, order
}

func (xnode *XNode) parseBody(node *Node) (string, bool) {
	if data, ok := xnode.getBodyData(node); ok {
		return data, true
	}

	for _, child := range node.GetChildNodes() {
		if data, ok := xnode.getBodyData(child); ok {
			return data, true
		}
	}

	return "", false
}

func (xnode *XNode) getBodyData(child *Node) (string, bool) {
	if child == nil {
		return "", false
	}
	if child.GetNodeType() == CDATA_SECTION_NODE || child.GetNodeType() == TEXT_NODE {
		return PropertyParser.Parse(child.GetNodeValue(), xnode.variables), true
	}
	return "", false
}
