package parsing

import (
	"strings"
	"testing"
)

func TestXPathParserShouldUseCurrentTreeOrderAfterCloneInsertionLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `
		<root>
		  <target><after /></target>
		  <fragment><inserted /></fragment>
		</root>
	`, "clone_insertion_order.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}

	targetNodes := root.childElementsByName("target")
	fragmentNodes := root.childElementsByName("fragment")
	if len(targetNodes) != 1 || len(fragmentNodes) != 1 {
		t.Fatalf("target/fragment counts = (%d, %d), want (1, 1)", len(targetNodes), len(fragmentNodes))
	}

	target := targetNodes[0]
	fragment := fragmentNodes[0]
	inserted := fragment.GetFirstChild()
	if inserted == nil {
		t.Fatal("fragment child should not be nil")
	}
	clonedInserted := inserted.CloneNode(true)
	if clonedInserted == nil {
		t.Fatal("cloned fragment child should not be nil")
	}

	if target.InsertBefore(clonedInserted, target.GetFirstChild()) == nil {
		t.Fatal("InsertBefore should succeed")
	}

	parser := mustNewXPathParserFromDocument(t, document, XPathParserConfig{Resource: "clone_insertion_order.xml"})
	nodes := parser.EvalNodes("/root/target/*")
	if len(nodes) != 2 {
		t.Fatalf("target child count = %d, want 2", len(nodes))
	}
	if actual := []string{nodes[0].GetName(), nodes[1].GetName()}; actual[0] != "inserted" || actual[1] != "after" {
		t.Fatalf("target child order = %v, want [inserted after]", actual)
	}
}

func TestXPathParserShouldUseCurrentTreeOrderAfterImportInsertionLikeDOM(t *testing.T) {
	destinationDocument := mustParseTestDocument(t, `<root><after /></root>`, "import_destination.xml")
	sourceDocument := mustParseTestDocument(t, `<root><imported /></root>`, "import_source.xml")

	destinationRoot := destinationDocument.GetDocumentElement()
	sourceRoot := sourceDocument.GetDocumentElement()
	if destinationRoot == nil || sourceRoot == nil {
		t.Fatal("document elements should not be nil")
	}

	sourceChild := sourceRoot.GetFirstChild()
	if sourceChild == nil {
		t.Fatal("source child should not be nil")
	}
	importedChild := destinationDocument.ImportNode(sourceChild, true)
	if importedChild == nil {
		t.Fatal("ImportNode should return a node")
	}

	if destinationRoot.InsertBefore(importedChild, destinationRoot.GetFirstChild()) == nil {
		t.Fatal("InsertBefore should succeed")
	}

	parser := mustNewXPathParserFromDocument(t, destinationDocument, XPathParserConfig{Resource: "import_destination.xml"})
	nodes := parser.EvalNodes("/root/*")
	if len(nodes) != 2 {
		t.Fatalf("root child count = %d, want 2", len(nodes))
	}
	if actual := []string{nodes[0].GetName(), nodes[1].GetName()}; actual[0] != "imported" || actual[1] != "after" {
		t.Fatalf("root child order = %v, want [imported after]", actual)
	}
}

func TestDocumentNodeShouldNotExposeOwnerDocumentLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root><child /></root>`, "document_owner_document.xml")

	documentNode := document.GetNode()
	if documentNode == nil {
		t.Fatal("document node should not be nil")
	}
	if documentNode.GetOwnerDocument() != nil {
		t.Fatal("document node ownerDocument should be nil like DOM")
	}

	documentElement := document.GetDocumentElement()
	if documentElement == nil {
		t.Fatal("document element should not be nil")
	}
	if documentElement.GetOwnerDocument() != document {
		t.Fatal("document element ownerDocument should still point to the document")
	}

	parser := mustNewXPathParserFromDocument(t, document, XPathParserConfig{Resource: "document_owner_document.xml"})
	if parser.EvalNode("/root/child") == nil {
		t.Fatal("XPath evaluation from document root should still work")
	}
}

func TestAttributeCloneShouldRetainValueChildrenEvenWhenShallowLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root attr="value" />`, "attribute_clone.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}

	attribute := root.GetAttribute("attr")
	if attribute == nil {
		t.Fatal("attribute should not be nil")
	}

	shallowClone := attribute.CloneNode(false)
	if shallowClone == nil {
		t.Fatal("shallow clone should not be nil")
	}
	if shallowClone.GetNodeType() != ATTRIBUTE_NODE {
		t.Fatalf("shallow clone node type = %d, want %d", shallowClone.GetNodeType(), ATTRIBUTE_NODE)
	}
	if len(shallowClone.GetChildNodes()) != 1 {
		t.Fatalf("shallow clone child count = %d, want 1", len(shallowClone.GetChildNodes()))
	}
	if shallowClone.GetFirstChild() == nil || shallowClone.GetFirstChild().GetNodeValue() != "value" {
		t.Fatalf("shallow clone first child = %#v, want text child %q", shallowClone.GetFirstChild(), "value")
	}

	deepClone := attribute.CloneNode(true)
	if deepClone == nil {
		t.Fatal("deep clone should not be nil")
	}
	if len(deepClone.GetChildNodes()) != 1 {
		t.Fatalf("deep clone child count = %d, want 1", len(deepClone.GetChildNodes()))
	}
	if deepClone.GetFirstChild() == nil || deepClone.GetFirstChild().GetNodeValue() != "value" {
		t.Fatalf("deep clone first child = %#v, want text child %q", deepClone.GetFirstChild(), "value")
	}
}

func TestSetNodeValueShouldIgnoreDocumentAndElementNodesLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root attr="value">text</root>`, "set_node_value.xml")

	documentNode := document.GetNode()
	if documentNode == nil {
		t.Fatal("document node should not be nil")
	}
	documentNode.SetNodeValue("changed-document")
	if actual := documentNode.GetNodeValue(); actual != "" {
		t.Fatalf("document node value = %q, want empty", actual)
	}

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}
	root.SetNodeValue("changed-root")
	if actual := root.GetNodeValue(); actual != "" {
		t.Fatalf("element node value = %q, want empty", actual)
	}

	attribute := root.GetAttribute("attr")
	if attribute == nil {
		t.Fatal("attribute should not be nil")
	}
	attribute.SetNodeValue("changed-attr")
	if actual := attribute.GetNodeValue(); actual != "changed-attr" {
		t.Fatalf("attribute node value = %q, want %q", actual, "changed-attr")
	}
	if attribute.GetFirstChild() == nil || attribute.GetFirstChild().GetNodeValue() != "changed-attr" {
		t.Fatalf("attribute child node value = %#v, want %q", attribute.GetFirstChild(), "changed-attr")
	}
}

func TestAttributeTextChildSetNodeValueShouldUpdateParentAttributeLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root attr="value" />`, "attribute_text_child_set_node_value.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}

	attribute := root.GetAttribute("attr")
	if attribute == nil {
		t.Fatal("attribute should not be nil")
	}
	textChild := attribute.GetFirstChild()
	if textChild == nil {
		t.Fatal("attribute text child should not be nil")
	}

	textChild.SetNodeValue("changed-child")

	if actual := textChild.GetNodeValue(); actual != "changed-child" {
		t.Fatalf("text child node value = %q, want %q", actual, "changed-child")
	}
	if actual := attribute.GetNodeValue(); actual != "changed-child" {
		t.Fatalf("attribute node value = %q, want %q", actual, "changed-child")
	}
}

func TestDocumentImportNodeShouldRejectDocumentNodesLikeDOM(t *testing.T) {
	sourceDocument := mustParseTestDocument(t, `<source><child /></source>`, "document_import_source.xml")
	destinationDocument := mustParseTestDocument(t, `<destination />`, "document_import_destination.xml")

	if imported := destinationDocument.ImportNode(sourceDocument.GetNode(), true); imported != nil {
		t.Fatalf("ImportNode(document) = %#v, want nil", imported)
	}
}

func TestDocumentCloneShouldOwnClonedChildrenLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root><child /></root>`, "document_clone.xml")

	shallowClone := document.GetNode().CloneNode(false)
	if shallowClone == nil {
		t.Fatal("shallow document clone should not be nil")
	}
	if shallowClone.GetNodeType() != DOCUMENT_NODE {
		t.Fatalf("shallow clone node type = %d, want %d", shallowClone.GetNodeType(), DOCUMENT_NODE)
	}
	if shallowClone.GetOwnerDocument() != nil {
		t.Fatal("shallow document clone ownerDocument should be nil like DOM")
	}
	if len(shallowClone.GetChildNodes()) != 0 {
		t.Fatalf("shallow document clone child count = %d, want 0", len(shallowClone.GetChildNodes()))
	}

	deepClone := document.GetNode().CloneNode(true)
	if deepClone == nil {
		t.Fatal("deep document clone should not be nil")
	}
	if len(deepClone.GetChildNodes()) != 1 {
		t.Fatalf("deep document clone child count = %d, want 1", len(deepClone.GetChildNodes()))
	}

	cloneRoot := deepClone.GetFirstChild()
	if cloneRoot == nil {
		t.Fatal("deep document clone root should not be nil")
	}
	cloneChild := cloneRoot.GetFirstChild()
	if cloneChild == nil {
		t.Fatal("deep document clone child should not be nil")
	}
	if cloneRoot.GetOwnerDocument() == document {
		t.Fatal("deep document clone root ownerDocument should not point to the original document")
	}
	if cloneRoot.GetOwnerDocument() == nil {
		t.Fatal("deep document clone root ownerDocument should not be nil")
	}
	if cloneRoot.GetOwnerDocument() != cloneChild.GetOwnerDocument() {
		t.Fatal("deep document clone descendants should share the cloned ownerDocument")
	}
}

func TestAttributeChildMutationsShouldUpdateParentAttributeValueLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root attr="value" />`, "attribute_child_mutations.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}
	attribute := root.GetAttribute("attr")
	if attribute == nil {
		t.Fatal("attribute should not be nil")
	}

	originalChild := attribute.GetFirstChild()
	if originalChild == nil {
		t.Fatal("attribute child should not be nil")
	}
	if removed := attribute.RemoveChild(originalChild); removed != originalChild {
		t.Fatalf("RemoveChild() = %#v, want %#v", removed, originalChild)
	}
	if actual := attribute.GetNodeValue(); actual != "" {
		t.Fatalf("attribute node value after remove = %q, want empty", actual)
	}

	insertedChild := originalChild.CloneNode(true)
	insertedChild.SetNodeValue("before")
	if inserted := attribute.InsertBefore(insertedChild, nil); inserted != insertedChild {
		t.Fatalf("InsertBefore() = %#v, want %#v", inserted, insertedChild)
	}
	if actual := attribute.GetNodeValue(); actual != "before" {
		t.Fatalf("attribute node value after insert = %q, want %q", actual, "before")
	}

	replacementChild := insertedChild.CloneNode(true)
	replacementChild.SetNodeValue("replaced")
	if replaced := attribute.ReplaceChild(replacementChild, insertedChild); replaced != insertedChild {
		t.Fatalf("ReplaceChild() = %#v, want %#v", replaced, insertedChild)
	}
	if actual := attribute.GetNodeValue(); actual != "replaced" {
		t.Fatalf("attribute node value after replace = %q, want %q", actual, "replaced")
	}
}

func TestAttributeSetNodeValueShouldReplaceChildrenWithSingleTextNodeLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root attr="value" />`, "attribute_set_node_value_children.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}
	attribute := root.GetAttribute("attr")
	if attribute == nil {
		t.Fatal("attribute should not be nil")
	}

	originalChild := attribute.GetFirstChild()
	if originalChild == nil {
		t.Fatal("attribute child should not be nil")
	}
	attribute.RemoveChild(originalChild)
	if len(attribute.GetChildNodes()) != 0 {
		t.Fatalf("attribute child count after remove = %d, want 0", len(attribute.GetChildNodes()))
	}

	attribute.SetNodeValue("changed")
	if actual := attribute.GetNodeValue(); actual != "changed" {
		t.Fatalf("attribute node value after set = %q, want %q", actual, "changed")
	}
	if len(attribute.GetChildNodes()) != 1 {
		t.Fatalf("attribute child count after set = %d, want 1", len(attribute.GetChildNodes()))
	}
	if attribute.GetFirstChild() == nil || attribute.GetFirstChild().GetNodeValue() != "changed" {
		t.Fatalf("attribute first child after set = %#v, want text child %q", attribute.GetFirstChild(), "changed")
	}

	prefixChild := originalChild.CloneNode(true)
	prefixChild.SetNodeValue("prefix")
	if inserted := attribute.InsertBefore(prefixChild, attribute.GetFirstChild()); inserted != prefixChild {
		t.Fatalf("InsertBefore() = %#v, want %#v", inserted, prefixChild)
	}
	if actual := attribute.GetNodeValue(); actual != "prefixchanged" {
		t.Fatalf("attribute node value before reset = %q, want %q", actual, "prefixchanged")
	}
	if len(attribute.GetChildNodes()) != 2 {
		t.Fatalf("attribute child count before reset = %d, want 2", len(attribute.GetChildNodes()))
	}

	attribute.SetNodeValue("reset")
	if actual := attribute.GetNodeValue(); actual != "reset" {
		t.Fatalf("attribute node value after reset = %q, want %q", actual, "reset")
	}
	if len(attribute.GetChildNodes()) != 1 {
		t.Fatalf("attribute child count after reset = %d, want 1", len(attribute.GetChildNodes()))
	}
	if attribute.GetFirstChild() == nil || attribute.GetFirstChild().GetNodeValue() != "reset" {
		t.Fatalf("attribute first child after reset = %#v, want text child %q", attribute.GetFirstChild(), "reset")
	}
}

func TestNodeInsertBeforeShouldMoveExistingEarlierSiblingJustBeforeReferenceLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root><a /><b /><c /></root>`, "insert_before_same_parent.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}

	children := root.elementChildren()
	if len(children) != 3 {
		t.Fatalf("root child count = %d, want 3", len(children))
	}

	moved := children[0]
	refChild := children[2]
	if inserted := root.InsertBefore(moved, refChild); inserted != moved {
		t.Fatalf("InsertBefore() = %#v, want moved child %#v", inserted, moved)
	}

	if actual := childElementNames(root); actual[0] != "b" || actual[1] != "a" || actual[2] != "c" {
		t.Fatalf("root child order = %v, want [b a c]", actual)
	}
	if moved.GetParentNode() != root {
		t.Fatal("moved child parent should remain root")
	}
}

func TestNodeReplaceChildShouldMoveExistingEarlierSiblingWithoutPanickingLikeDOM(t *testing.T) {
	document := mustParseTestDocument(t, `<root><a /><b /><c /></root>`, "replace_child_same_parent.xml")

	root := document.GetDocumentElement()
	if root == nil {
		t.Fatal("document element should not be nil")
	}

	children := root.elementChildren()
	if len(children) != 3 {
		t.Fatalf("root child count = %d, want 3", len(children))
	}

	newChild := children[0]
	oldChild := children[2]
	if replaced := root.ReplaceChild(newChild, oldChild); replaced != oldChild {
		t.Fatalf("ReplaceChild() = %#v, want old child %#v", replaced, oldChild)
	}

	if actual := childElementNames(root); actual[0] != "b" || actual[1] != "a" {
		t.Fatalf("root child order = %v, want [b a]", actual)
	}
	if newChild.GetParentNode() != root {
		t.Fatal("new child parent should remain root")
	}
	if oldChild.GetParentNode() != nil {
		t.Fatal("replaced child parent should be nil")
	}
}

func mustParseTestDocument(t *testing.T, xmlText string, resource string) *Document {
	t.Helper()

	document, err := ParseDocument(strings.NewReader(xmlText), resource)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}
	return document
}

func mustNewXPathParserFromDocument(t *testing.T, document *Document, configs ...XPathParserConfig) *XPathParser {
	t.Helper()

	parser, err := NewXPathParserFromDocument(document, configs...)
	if err != nil {
		t.Fatalf("NewXPathParserFromDocument failed: %v", err)
	}
	return parser
}

func childElementNames(node *Node) []string {
	if node == nil {
		return nil
	}
	children := node.elementChildren()
	names := make([]string, 0, len(children))
	for _, child := range children {
		names = append(names, child.GetNodeName())
	}
	return names
}
