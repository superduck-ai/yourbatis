package parsing

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode/utf32"
)

type NodeType int

const (
	ELEMENT_NODE          NodeType = 1
	ATTRIBUTE_NODE        NodeType = 2
	TEXT_NODE             NodeType = 3
	CDATA_SECTION_NODE    NodeType = 4
	ENTITY_REFERENCE_NODE NodeType = 5
	DOCUMENT_NODE         NodeType = 9
)

type Position struct {
	Source string
	Line   int
	Column int
	Offset int64
}

type DocumentType struct {
	Present  bool
	RootName string
	PublicID string
	SystemID string
}

type Document struct {
	node         *Node
	source       string
	sequence     int
	documentType DocumentType
}

type Node struct {
	nodeType      NodeType
	nodeName      string
	nodeValue     string
	parent        *Node
	children      []*Node
	attributes    []*Node
	ownerDocument *Document
	position      Position
	order         int
}

type ParseDocumentConfig struct {
	Validation          bool
	EntityResolver      EntityResolver
	InputAlreadyDecoded bool
}

type xmlDeclaredEntity struct {
	name        string
	value       string
	placeholder string
}

type xmlDeclaredEntityRegistry struct {
	byName        map[string]xmlDeclaredEntity
	byPlaceholder map[string]xmlDeclaredEntity
	nextID        int
}

type xmlCharacterDataSegment struct {
	text       string
	entityName string
	isEntity   bool
}

const (
	xmlDeclaredEntityPlaceholderStart = "\uE000"
	xmlDeclaredEntityPlaceholderEnd   = "\uE001"
)

var xmlDeclaredEntityPattern = regexp.MustCompile(
	`(?is)<!ENTITY\s+([A-Za-z_:][A-Za-z0-9_.:-]*)\s+"([^"]*)"\s*>|<!ENTITY\s+([A-Za-z_:][A-Za-z0-9_.:-]*)\s+'([^']*)'\s*>`,
)

var xmlEncodingDeclarationPattern = regexp.MustCompile(
	`(?:^|[ \t\r\n])encoding[ \t\r\n]*=[ \t\r\n]*(?:"([^"]*)"|'([^']*)')`,
)

func ParseDocument(reader io.Reader, source string, configs ...ParseDocumentConfig) (*Document, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewParsingException(fmt.Sprintf("Error creating document instance. Cause: %v", err), err)
	}

	document := newDocument(source)
	config := firstParseDocumentConfig(configs)
	data, err = decodeXMLInput(data, config.InputAlreadyDecoded)
	if err != nil {
		return nil, NewParsingException(fmt.Sprintf("Error creating document instance. Cause: %v", err), err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	decoder.CharsetReader = decodedXMLDeclarationPassthroughReader
	entityRegistry := newXMLDeclaredEntityRegistry()
	stack := []*Node{document.node}
	var previousOffset int64

	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, NewParsingException(fmt.Sprintf("Error creating document instance. Cause: %v", err), err)
		}

		currentOffset := decoder.InputOffset()
		line, column := decoder.InputPos()
		position := Position{
			Source: source,
			Line:   line,
			Column: column,
			Offset: currentOffset,
		}

		switch current := token.(type) {
		case xml.StartElement:
			element := document.newNode(ELEMENT_NODE, qualifiedXMLName(current.Name), "", position)
			for _, attribute := range current.Attr {
				attributeValue := attribute.Value
				if entityRegistry != nil {
					attributeValue = entityRegistry.resolveAttributeValue(attributeValue)
				}
				element.appendAttribute(document.newAttributeNode(qualifiedXMLName(attribute.Name), attributeValue, position))
			}
			stack[len(stack)-1].appendChild(element)
			stack = append(stack, element)
		case xml.EndElement:
			if len(stack) == 1 {
				return nil, NewParsingException("Error creating document instance. Cause: unexpected end element", nil)
			}
			if currentName, expectedName := qualifiedXMLName(current.Name), stack[len(stack)-1].GetNodeName(); currentName != expectedName {
				return nil, NewParsingException(
					fmt.Sprintf("Error creating document instance. Cause: end element %q does not match start element %q", currentName, expectedName),
					nil,
				)
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			text := string(current)
			nodeType := classifyCharacterDataNode(data, previousOffset, currentOffset)
			if text == "" && nodeType != CDATA_SECTION_NODE {
				break
			}
			if nodeType == TEXT_NODE && entityRegistry != nil && entityRegistry.hasPlaceholders(text) {
				for _, segment := range entityRegistry.splitCharacterData(text) {
					if segment.isEntity {
						stack[len(stack)-1].appendChild(
							document.newNode(ENTITY_REFERENCE_NODE, segment.entityName, "", position),
						)
						continue
					}
					if segment.text == "" {
						continue
					}
					stack[len(stack)-1].appendChild(
						document.newNode(TEXT_NODE, characterDataNodeName(TEXT_NODE), segment.text, position),
					)
				}
				break
			}
			stack[len(stack)-1].appendChild(document.newNode(nodeType, characterDataNodeName(nodeType), text, position))
		case xml.Comment:
			break
		case xml.Directive:
			if documentType, ok := parseDocumentTypeDirective(string(current)); ok {
				document.documentType = documentType
			}
			resolvedDTD, err := resolveDirective(current, config)
			if err != nil {
				return nil, NewParsingException(fmt.Sprintf("Error creating document instance. Cause: %v", err), err)
			}
			entityRegistry.registerDefinitions(string(resolvedDTD))
			entityRegistry.registerDefinitions(string(current))
			entityRegistry.applyToDecoder(decoder)
		case xml.ProcInst:
			break
		}
		previousOffset = currentOffset
	}

	if document.GetDocumentElement() == nil {
		return nil, NewParsingException("Error creating document instance. Cause: XML document does not contain a root element", nil)
	}
	if len(stack) != 1 {
		return nil, NewParsingException("Error creating document instance. Cause: XML document ended before all elements were closed", nil)
	}

	return document, nil
}

func decodedXMLDeclarationPassthroughReader(_ string, input io.Reader) (io.Reader, error) {
	// decodeXMLInput converts raw InputStream bytes before tokenization. The
	// declaration remains in the normalized document as metadata, so encoding/xml
	// must continue over the already-decoded UTF-8 stream without decoding it again.
	return input, nil
}

func normalizeXMLCharsetName(charset string) string {
	var builder strings.Builder
	for _, current := range strings.ToLower(strings.TrimSpace(charset)) {
		switch current {
		case '-', '_', ' ', ':':
			continue
		default:
			builder.WriteRune(current)
		}
	}
	return builder.String()
}

func decodeXMLInput(data []byte, inputAlreadyDecoded bool) ([]byte, error) {
	if inputAlreadyDecoded {
		return stripXMLUTF8BOM(data), nil
	}

	decoded, decodedEBCDIC, err := decodeXMLEBCDICInput(data)
	if err != nil {
		return nil, err
	}
	if decodedEBCDIC {
		return decoded, nil
	}

	utf32Encoding, _, _, _ := detectXMLUTF32Encoding(data)
	decoded, decodedUTF32, err := decodeXMLUTF32Input(decoded)
	if err != nil {
		return nil, err
	}
	if decodedUTF32 {
		if err := validateDetectedXMLEncoding(decoded, "utf32", utf32Encoding); err != nil {
			return nil, err
		}
		return decoded, nil
	}

	utf16Encoding, _, _, _ := detectXMLUTF16Encoding(data)
	decoded, decodedUTF16, err := decodeXMLUTF16Input(decoded)
	if err != nil {
		return nil, err
	}
	if decodedUTF16 {
		if err := validateDetectedXMLEncoding(decoded, "utf16", utf16Encoding); err != nil {
			return nil, err
		}
		return decoded, nil
	}

	declaredEncoding := findXMLEncodingDeclaration(decoded)
	switch normalizeXMLCharsetName(declaredEncoding) {
	case "", "utf8":
		return decoded, nil
	case "usascii", "ascii":
		for _, value := range decoded {
			if value > 0x7f {
				return nil, fmt.Errorf("invalid US-ASCII byte 0x%02X", value)
			}
		}
		return decoded, nil
	case "utf16", "utf16be", "utf16le":
		return nil, fmt.Errorf(
			"XML encoding declaration %q does not match the detected input encoding",
			declaredEncoding,
		)
	case "utf32", "utf32be", "utf32le":
		return nil, fmt.Errorf(
			"XML encoding declaration %q does not match the detected input encoding",
			declaredEncoding,
		)
	default:
		return decodeXMLWithDeclaredEncoding(decoded, declaredEncoding)
	}
}

func decodeXMLEBCDICInput(data []byte) ([]byte, bool, error) {
	if len(data) < 4 || !bytes.Equal(data[:4], []byte{0x4c, 0x6f, 0xa7, 0x94}) {
		return data, false, nil
	}

	bootstrap, err := charmap.CodePage037.NewDecoder().Bytes(data)
	if err != nil {
		return nil, false, fmt.Errorf("invalid EBCDIC XML input: %w", err)
	}
	declaredEncoding := findXMLEncodingDeclaration(bootstrap)
	if declaredEncoding == "" {
		return nil, false, fmt.Errorf("XML encoding declaration is required for EBCDIC input")
	}

	decoded, err := decodeXMLWithDeclaredEncoding(data, declaredEncoding)
	if err != nil {
		return nil, false, err
	}
	return decoded, true, nil
}

func decodeXMLWithDeclaredEncoding(data []byte, declaredEncoding string) ([]byte, error) {
	charsetEncoding, lookupErr := ianaindex.IANA.Encoding(declaredEncoding)
	if lookupErr != nil || charsetEncoding == nil {
		return nil, fmt.Errorf("unsupported XML encoding %q", declaredEncoding)
	}
	decoded, decodeErr := charsetEncoding.NewDecoder().Bytes(data)
	if decodeErr != nil {
		return nil, fmt.Errorf("invalid %s XML input: %w", declaredEncoding, decodeErr)
	}
	return decoded, nil
}

func stripXMLUTF8BOM(data []byte) []byte {
	if len(data) >= 3 && bytes.Equal(data[:3], []byte{0xef, 0xbb, 0xbf}) {
		return data[3:]
	}
	return data
}

func validateDetectedXMLEncoding(data []byte, family, detectedEncoding string) error {
	declaredEncoding := findXMLEncodingDeclaration(data)
	normalizedEncoding := normalizeXMLCharsetName(declaredEncoding)
	if normalizedEncoding == "" ||
		normalizedEncoding == family ||
		normalizedEncoding == detectedEncoding {
		return nil
	}
	return fmt.Errorf(
		"XML encoding declaration %q conflicts with detected %s input",
		declaredEncoding,
		strings.ToUpper(detectedEncoding),
	)
}

func findXMLEncodingDeclaration(data []byte) string {
	if !bytes.HasPrefix(data, []byte("<?xml")) {
		return ""
	}
	declarationEnd := bytes.Index(data, []byte("?>"))
	if declarationEnd < 0 {
		return ""
	}

	match := xmlEncodingDeclarationPattern.FindSubmatch(data[:declarationEnd])
	if len(match) == 0 {
		return ""
	}
	if len(match[1]) > 0 {
		return string(match[1])
	}
	return string(match[2])
}

func decodeXMLUTF16Input(data []byte) ([]byte, bool, error) {
	data = stripXMLUTF8BOM(data)

	_, byteOrder, offset, ok := detectXMLUTF16Encoding(data)
	if !ok {
		return data, false, nil
	}

	encoded := data[offset:]
	if len(encoded)%2 != 0 {
		return nil, false, fmt.Errorf("invalid UTF-16 XML input: odd byte length %d", len(encoded))
	}

	var builder strings.Builder
	builder.Grow(len(encoded))
	for index := 0; index < len(encoded); index += 2 {
		current := byteOrder.Uint16(encoded[index : index+2])
		switch {
		case current >= 0xd800 && current <= 0xdbff:
			if index+4 > len(encoded) {
				return nil, false, fmt.Errorf(
					"invalid UTF-16 XML input: unpaired high surrogate 0x%04X",
					current,
				)
			}
			next := byteOrder.Uint16(encoded[index+2 : index+4])
			if next < 0xdc00 || next > 0xdfff {
				return nil, false, fmt.Errorf(
					"invalid UTF-16 XML input: high surrogate 0x%04X is followed by 0x%04X",
					current,
					next,
				)
			}
			builder.WriteRune(utf16.DecodeRune(rune(current), rune(next)))
			index += 2
		case current >= 0xdc00 && current <= 0xdfff:
			return nil, false, fmt.Errorf(
				"invalid UTF-16 XML input: unpaired low surrogate 0x%04X",
				current,
			)
		default:
			builder.WriteRune(rune(current))
		}
	}
	return []byte(builder.String()), true, nil
}

func decodeXMLUTF32Input(data []byte) ([]byte, bool, error) {
	_, endianness, offset, ok := detectXMLUTF32Encoding(data)
	if !ok {
		return data, false, nil
	}

	encoded := data[offset:]
	if err := validateXMLUTF32CodeUnits(encoded, endianness); err != nil {
		return nil, false, err
	}
	decoded, err := utf32.UTF32(endianness, utf32.IgnoreBOM).
		NewDecoder().
		Bytes(encoded)
	if err != nil {
		return nil, false, fmt.Errorf("invalid UTF-32 XML input: %w", err)
	}
	return decoded, true, nil
}

func validateXMLUTF32CodeUnits(data []byte, endianness utf32.Endianness) error {
	if len(data)%4 != 0 {
		return fmt.Errorf("invalid UTF-32 XML input: byte length %d is not a multiple of 4", len(data))
	}

	var byteOrder binary.ByteOrder = binary.BigEndian
	if endianness == utf32.LittleEndian {
		byteOrder = binary.LittleEndian
	}
	for index := 0; index < len(data); index += 4 {
		codePoint := byteOrder.Uint32(data[index : index+4])
		if codePoint > 0x10ffff || codePoint >= 0xd800 && codePoint <= 0xdfff {
			return fmt.Errorf(
				"invalid UTF-32 XML input: invalid code point U+%04X at byte offset %d",
				codePoint,
				index,
			)
		}
	}
	return nil
}

func detectXMLUTF32Encoding(data []byte) (string, utf32.Endianness, int, bool) {
	switch {
	case len(data) >= 4 &&
		data[0] == 0x00 && data[1] == 0x00 && data[2] == 0xfe && data[3] == 0xff:
		return "utf32be", utf32.BigEndian, 4, true
	case len(data) >= 4 &&
		data[0] == 0xff && data[1] == 0xfe && data[2] == 0x00 && data[3] == 0x00:
		return "utf32le", utf32.LittleEndian, 4, true
	case len(data) >= 4 &&
		data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x00 && data[3] == 0x3c:
		return "utf32be", utf32.BigEndian, 0, true
	case len(data) >= 4 &&
		data[0] == 0x3c && data[1] == 0x00 && data[2] == 0x00 && data[3] == 0x00:
		return "utf32le", utf32.LittleEndian, 0, true
	default:
		return "", utf32.BigEndian, 0, false
	}
}

func detectXMLUTF16Encoding(data []byte) (string, binary.ByteOrder, int, bool) {
	switch {
	case len(data) >= 2 && data[0] == 0xfe && data[1] == 0xff:
		return "utf16be", binary.BigEndian, 2, true
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xfe:
		return "utf16le", binary.LittleEndian, 2, true
	case len(data) >= 4 && data[0] == 0x00 && data[1] == 0x3c && data[2] == 0x00 && data[3] == 0x3f:
		return "utf16be", binary.BigEndian, 0, true
	case len(data) >= 4 && data[0] == 0x3c && data[1] == 0x00 && data[2] == 0x3f && data[3] == 0x00:
		return "utf16le", binary.LittleEndian, 0, true
	default:
		return "", nil, 0, false
	}
}

func qualifiedXMLName(name xml.Name) string {
	if name.Space == "" || name.Local == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

func classifyCharacterDataNode(data []byte, startOffset, endOffset int64) NodeType {
	if startOffset < 0 || endOffset < startOffset || endOffset > int64(len(data)) {
		return TEXT_NODE
	}
	if bytes.Contains(data[startOffset:endOffset], []byte("<![CDATA[")) {
		return CDATA_SECTION_NODE
	}
	return TEXT_NODE
}

func characterDataNodeName(nodeType NodeType) string {
	if nodeType == CDATA_SECTION_NODE {
		return "#cdata-section"
	}
	return "#text"
}

func firstParseDocumentConfig(configs []ParseDocumentConfig) ParseDocumentConfig {
	if len(configs) == 0 {
		return ParseDocumentConfig{}
	}
	return configs[0]
}

func newXMLDeclaredEntityRegistry() *xmlDeclaredEntityRegistry {
	return &xmlDeclaredEntityRegistry{
		byName:        make(map[string]xmlDeclaredEntity),
		byPlaceholder: make(map[string]xmlDeclaredEntity),
	}
}

func (registry *xmlDeclaredEntityRegistry) registerDefinitions(dtdText string) {
	if registry == nil || dtdText == "" {
		return
	}
	for _, match := range xmlDeclaredEntityPattern.FindAllStringSubmatch(dtdText, -1) {
		name := match[1]
		value := match[2]
		if name == "" {
			name = match[3]
			value = match[4]
		}
		if name == "" {
			continue
		}
		placeholder := registry.newPlaceholder()
		if existing, ok := registry.byName[name]; ok {
			delete(registry.byPlaceholder, existing.placeholder)
		}
		entity := xmlDeclaredEntity{
			name:        name,
			value:       value,
			placeholder: placeholder,
		}
		registry.byName[name] = entity
		registry.byPlaceholder[placeholder] = entity
	}
}

func (registry *xmlDeclaredEntityRegistry) newPlaceholder() string {
	if registry == nil {
		return ""
	}
	placeholder := xmlDeclaredEntityPlaceholderStart + strconv.Itoa(registry.nextID) + xmlDeclaredEntityPlaceholderEnd
	registry.nextID++
	return placeholder
}

func (registry *xmlDeclaredEntityRegistry) applyToDecoder(decoder *xml.Decoder) {
	if registry == nil || decoder == nil || len(registry.byName) == 0 {
		return
	}
	if decoder.Entity == nil {
		decoder.Entity = make(map[string]string)
	}
	for name, entity := range registry.byName {
		decoder.Entity[name] = entity.placeholder
	}
}

func (registry *xmlDeclaredEntityRegistry) hasPlaceholders(value string) bool {
	return registry != nil &&
		value != "" &&
		len(registry.byPlaceholder) > 0 &&
		strings.Contains(value, xmlDeclaredEntityPlaceholderStart)
}

func (registry *xmlDeclaredEntityRegistry) resolveAttributeValue(value string) string {
	if !registry.hasPlaceholders(value) {
		return value
	}
	for placeholder, entity := range registry.byPlaceholder {
		value = strings.ReplaceAll(value, placeholder, entity.value)
	}
	return value
}

func (registry *xmlDeclaredEntityRegistry) splitCharacterData(value string) []xmlCharacterDataSegment {
	if !registry.hasPlaceholders(value) {
		if value == "" {
			return nil
		}
		return []xmlCharacterDataSegment{{text: value}}
	}

	segments := make([]xmlCharacterDataSegment, 0)
	for value != "" {
		start := strings.Index(value, xmlDeclaredEntityPlaceholderStart)
		if start < 0 {
			if value != "" {
				segments = append(segments, xmlCharacterDataSegment{text: value})
			}
			break
		}
		if start > 0 {
			segments = append(segments, xmlCharacterDataSegment{text: value[:start]})
		}

		remaining := value[start:]
		endIndex := strings.Index(remaining[len(xmlDeclaredEntityPlaceholderStart):], xmlDeclaredEntityPlaceholderEnd)
		if endIndex < 0 {
			segments = append(segments, xmlCharacterDataSegment{text: remaining})
			break
		}

		placeholderEnd := len(xmlDeclaredEntityPlaceholderStart) + endIndex + len(xmlDeclaredEntityPlaceholderEnd)
		placeholder := remaining[:placeholderEnd]
		if entity, ok := registry.byPlaceholder[placeholder]; ok {
			segments = append(segments, xmlCharacterDataSegment{
				entityName: entity.name,
				isEntity:   true,
			})
		} else {
			segments = append(segments, xmlCharacterDataSegment{text: placeholder})
		}
		value = remaining[placeholderEnd:]
	}
	return segments
}

func resolveDirective(directive xml.Directive, config ParseDocumentConfig) ([]byte, error) {
	if config.EntityResolver == nil {
		return nil, nil
	}

	documentType, ok := parseDocumentTypeDirective(string(directive))
	if !ok || documentType.SystemID == "" {
		return nil, nil
	}

	return config.EntityResolver.ResolveEntity(documentType.PublicID, documentType.SystemID)
}

func parseDocumentTypeDirective(directive string) (documentType DocumentType, ok bool) {
	directive = strings.TrimSpace(directive)
	if directive == "" {
		return DocumentType{}, false
	}

	upperDirective := strings.ToUpper(directive)
	if !strings.HasPrefix(upperDirective, "DOCTYPE") {
		return DocumentType{}, false
	}

	documentType = DocumentType{Present: true}
	remainder := strings.TrimSpace(directive[len("DOCTYPE"):])
	if remainder == "" {
		return documentType, true
	}

	var directiveKeyword string
	documentType.RootName, directiveKeyword = parseDocumentTypeDeclaration(remainder)

	quotedValues := extractQuotedValues(directive)
	switch directiveKeyword {
	case "PUBLIC":
		if len(quotedValues) >= 2 {
			documentType.PublicID = quotedValues[0]
			documentType.SystemID = quotedValues[1]
		}
	case "SYSTEM":
		if len(quotedValues) >= 1 {
			documentType.SystemID = quotedValues[0]
		}
	}
	return documentType, true
}

func parseDocumentTypeDeclaration(remainder string) (rootName string, directiveKeyword string) {
	rootEnd := len(remainder)
	for index := 0; index < len(remainder); index++ {
		switch remainder[index] {
		case ' ', '\t', '\r', '\n', '[', '>':
			rootEnd = index
			index = len(remainder)
		}
	}
	rootName = remainder[:rootEnd]
	if rootEnd >= len(remainder) {
		return rootName, ""
	}

	remainder = strings.TrimSpace(remainder[rootEnd:])
	if remainder == "" || remainder[0] == '[' || remainder[0] == '>' {
		return rootName, ""
	}

	keywordEnd := len(remainder)
	for index := 0; index < len(remainder); index++ {
		switch remainder[index] {
		case ' ', '\t', '\r', '\n', '[', '>':
			keywordEnd = index
			index = len(remainder)
		}
	}
	return rootName, strings.ToUpper(remainder[:keywordEnd])
}

func extractQuotedValues(text string) []string {
	values := make([]string, 0)
	for index := 0; index < len(text); index++ {
		quote := text[index]
		if quote != '"' && quote != '\'' {
			continue
		}

		start := index + 1
		index++
		for index < len(text) {
			if text[index] == quote && text[index-1] != '\\' {
				break
			}
			index++
		}
		if index <= len(text) {
			values = append(values, text[start:index])
		}
	}
	return values
}

func newDocument(source string) *Document {
	document := &Document{
		source: source,
	}
	document.node = &Node{
		nodeType:      DOCUMENT_NODE,
		nodeName:      "#document",
		ownerDocument: document,
		position: Position{
			Source: source,
		},
		order: document.nextOrder(),
	}
	return document
}

func (document *Document) nextOrder() int {
	document.sequence++
	return document.sequence
}

func (document *Document) newNode(nodeType NodeType, nodeName, nodeValue string, position Position) *Node {
	return &Node{
		nodeType:      nodeType,
		nodeName:      nodeName,
		nodeValue:     nodeValue,
		ownerDocument: document,
		position:      position,
		order:         document.nextOrder(),
	}
}

func (document *Document) newAttributeNode(nodeName, nodeValue string, position Position) *Node {
	attribute := document.newNode(ATTRIBUTE_NODE, nodeName, nodeValue, position)
	attribute.children = append(attribute.children, document.newNode(TEXT_NODE, "#text", nodeValue, position))
	attribute.children[0].parent = attribute
	return attribute
}

func (document *Document) GetNode() *Node {
	if document == nil {
		return nil
	}
	return document.node
}

func (document *Document) GetDocumentElement() *Node {
	if document == nil || document.node == nil {
		return nil
	}
	for _, child := range document.node.children {
		if child.nodeType == ELEMENT_NODE {
			return child
		}
	}
	return nil
}

func (document *Document) GetDocumentType() DocumentType {
	if document == nil {
		return DocumentType{}
	}
	return document.documentType
}

func (document *Document) ImportNode(node *Node, deep bool) *Node {
	if document == nil || node == nil || node.nodeType == DOCUMENT_NODE {
		return nil
	}
	clone := node.CloneNode(deep)
	document.adopt(clone, nil)
	return clone
}

func (document *Document) adopt(node *Node, parent *Node) {
	if node == nil {
		return
	}
	node.parent = parent
	node.ownerDocument = document
	for _, attribute := range node.attributes {
		document.adopt(attribute, node)
	}
	for _, child := range node.children {
		document.adopt(child, node)
	}
}

func (node *Node) GetNodeType() NodeType {
	if node == nil {
		return 0
	}
	return node.nodeType
}

func (node *Node) GetNodeName() string {
	if node == nil {
		return ""
	}
	return node.nodeName
}

func (node *Node) GetNodeValue() string {
	if node == nil {
		return ""
	}
	return node.nodeValue
}

func (node *Node) SetNodeValue(value string) {
	if node == nil {
		return
	}
	switch node.nodeType {
	case ATTRIBUTE_NODE, TEXT_NODE, CDATA_SECTION_NODE:
	default:
		return
	}
	node.nodeValue = value
	if node.nodeType == ATTRIBUTE_NODE {
		node.replaceAttributeValueChildren(value)
		return
	}
	if (node.nodeType == TEXT_NODE || node.nodeType == CDATA_SECTION_NODE) && node.parent != nil && node.parent.nodeType == ATTRIBUTE_NODE {
		node.parent.nodeValue = node.parent.attributeChildValue()
	}
}

func (node *Node) GetParentNode() *Node {
	if node == nil {
		return nil
	}
	if node.nodeType == ATTRIBUTE_NODE {
		return nil
	}
	return node.parent
}

func (node *Node) GetOwnerDocument() *Document {
	if node == nil {
		return nil
	}
	if node.nodeType == DOCUMENT_NODE {
		return nil
	}
	return node.ownerDocument
}

func (node *Node) GetChildNodes() []*Node {
	if node == nil {
		return nil
	}
	return node.children
}

func (node *Node) GetAttributes() []*Node {
	if node == nil {
		return nil
	}
	return node.attributes
}

func (node *Node) GetAttribute(name string) *Node {
	if node == nil {
		return nil
	}
	for _, attribute := range node.attributes {
		if attribute.nodeName == name {
			return attribute
		}
	}
	return nil
}

func (node *Node) HasChildNodes() bool {
	return node != nil && len(node.children) > 0
}

func (node *Node) GetFirstChild() *Node {
	if node == nil || len(node.children) == 0 {
		return nil
	}
	return node.children[0]
}

func (node *Node) GetPosition() Position {
	if node == nil {
		return Position{}
	}
	return node.position
}

func (node *Node) ReplaceChild(newChild, oldChild *Node) *Node {
	if node == nil || newChild == nil || oldChild == nil {
		return nil
	}
	if node.indexOfChild(oldChild) < 0 {
		return nil
	}
	if newChild == oldChild {
		return oldChild
	}
	newChild.detach()
	index := node.indexOfChild(oldChild)
	if index < 0 {
		return nil
	}
	node.attachChild(index, newChild)
	oldChild.parent = nil
	node.syncAttributeValueFromChildren()
	return oldChild
}

func (node *Node) InsertBefore(newChild, refChild *Node) *Node {
	if node == nil || newChild == nil {
		return nil
	}
	if refChild == nil {
		node.appendChild(newChild)
		return newChild
	}
	if node.indexOfChild(refChild) < 0 {
		return nil
	}
	if newChild == refChild {
		return newChild
	}
	newChild.detach()
	index := node.indexOfChild(refChild)
	if index < 0 {
		return nil
	}
	node.children = append(node.children, nil)
	copy(node.children[index+1:], node.children[index:])
	node.attachChild(index, newChild)
	node.syncAttributeValueFromChildren()
	return newChild
}

func (node *Node) RemoveChild(child *Node) *Node {
	if node == nil || child == nil {
		return nil
	}
	index := node.indexOfChild(child)
	if index < 0 {
		return nil
	}
	node.children = append(node.children[:index], node.children[index+1:]...)
	child.parent = nil
	node.syncAttributeValueFromChildren()
	return child
}

func (node *Node) CloneNode(deep bool) *Node {
	if node == nil {
		return nil
	}
	if node.nodeType == DOCUMENT_NODE {
		return node.cloneDocumentNode(deep)
	}

	cloneChildren := deep || node.nodeType == ATTRIBUTE_NODE
	clone := &Node{
		nodeType:      node.nodeType,
		nodeName:      node.nodeName,
		nodeValue:     node.nodeValue,
		ownerDocument: node.ownerDocument,
		position:      node.position,
		order:         node.order,
	}

	for _, attribute := range node.attributes {
		attributeClone := attribute.CloneNode(true)
		attributeClone.parent = clone
		clone.attributes = append(clone.attributes, attributeClone)
	}

	if cloneChildren {
		for _, child := range node.children {
			childClone := child.CloneNode(true)
			childClone.parent = clone
			clone.children = append(clone.children, childClone)
		}
	}

	return clone
}

func (node *Node) stringValue() string {
	if node == nil {
		return ""
	}

	switch node.nodeType {
	case ATTRIBUTE_NODE, TEXT_NODE, CDATA_SECTION_NODE:
		return node.nodeValue
	case ENTITY_REFERENCE_NODE:
		return ""
	case ELEMENT_NODE, DOCUMENT_NODE:
		value := ""
		for _, child := range node.children {
			value += child.stringValue()
		}
		return value
	default:
		return node.nodeValue
	}
}

func (node *Node) elementChildren() []*Node {
	if node == nil {
		return nil
	}
	children := make([]*Node, 0, len(node.children))
	for _, child := range node.children {
		if child.nodeType == ELEMENT_NODE {
			children = append(children, child)
		}
	}
	return children
}

func (node *Node) childElementsByName(name string) []*Node {
	if node == nil {
		return nil
	}
	children := make([]*Node, 0)
	for _, child := range node.children {
		if child.nodeType == ELEMENT_NODE && child.nodeName == name {
			children = append(children, child)
		}
	}
	return children
}

func (node *Node) appendChild(child *Node) {
	if node == nil || child == nil {
		return
	}
	child.detach()
	node.children = append(node.children, child)
	node.attachChild(len(node.children)-1, child)
	node.syncAttributeValueFromChildren()
}

func (node *Node) appendAttribute(attribute *Node) {
	if node == nil || attribute == nil {
		return
	}
	attribute.detach()
	attribute.parent = node
	attribute.ownerDocument = node.ownerDocument
	node.attributes = append(node.attributes, attribute)
	for _, child := range attribute.children {
		node.ownerDocument.adopt(child, attribute)
	}
}

func (node *Node) attachChild(index int, child *Node) {
	child.parent = node
	child.ownerDocument = node.ownerDocument
	node.children[index] = child
	for _, attribute := range child.attributes {
		child.ownerDocument.adopt(attribute, child)
	}
	for _, grandChild := range child.children {
		child.ownerDocument.adopt(grandChild, child)
	}
}

func (node *Node) detach() {
	if node == nil || node.parent == nil {
		return
	}
	if node.parent.indexOfChild(node) >= 0 {
		node.parent.RemoveChild(node)
		return
	}
	node.parent.removeAttribute(node)
}

func (node *Node) indexOfChild(child *Node) int {
	for index, current := range node.children {
		if current == child {
			return index
		}
	}
	return -1
}

func (node *Node) removeAttribute(attribute *Node) {
	for index, current := range node.attributes {
		if current == attribute {
			node.attributes = append(node.attributes[:index], node.attributes[index+1:]...)
			attribute.parent = nil
			return
		}
	}
}

func (node *Node) attributeChildValue() string {
	if node == nil {
		return ""
	}
	var builder strings.Builder
	for _, child := range node.children {
		if child == nil {
			continue
		}
		builder.WriteString(child.nodeValue)
	}
	return builder.String()
}

func (node *Node) syncAttributeValueFromChildren() {
	if node == nil || node.nodeType != ATTRIBUTE_NODE {
		return
	}
	node.nodeValue = node.attributeChildValue()
}

func (node *Node) replaceAttributeValueChildren(value string) {
	if node == nil || node.nodeType != ATTRIBUTE_NODE {
		return
	}

	var textChild *Node
	if node.ownerDocument != nil {
		textChild = node.ownerDocument.newNode(TEXT_NODE, characterDataNodeName(TEXT_NODE), value, node.position)
	} else {
		textChild = &Node{
			nodeType:  TEXT_NODE,
			nodeName:  characterDataNodeName(TEXT_NODE),
			nodeValue: value,
			position:  node.position,
		}
	}
	textChild.parent = node
	textChild.ownerDocument = node.ownerDocument
	node.children = []*Node{textChild}
}

func (node *Node) cloneDocumentNode(deep bool) *Node {
	if node == nil || node.nodeType != DOCUMENT_NODE {
		return nil
	}

	source := node.position.Source
	if node.ownerDocument != nil {
		source = node.ownerDocument.source
	}
	clonedDocument := newDocument(source)
	clonedDocument.node.position = node.position

	if deep {
		for _, child := range node.children {
			clonedDocument.node.appendChild(child.CloneNode(true))
		}
	}

	return clonedDocument.node
}
