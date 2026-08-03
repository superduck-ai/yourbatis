package parsing

import "testing"

func TestPropertyTokenizerShouldParseSimpleProperty(t *testing.T) {
	tokenizer := NewPropertyTokenizer("id")

	if actual := tokenizer.GetIndexedName(); actual != "id" {
		t.Fatalf("indexedName = %q, want %q", actual, "id")
	}
	if actual := tokenizer.GetName(); actual != "id" {
		t.Fatalf("name = %q, want %q", actual, "id")
	}
	if actual := tokenizer.GetChildren(); actual != "" {
		t.Fatalf("children = %q, want empty", actual)
	}
	if actual := tokenizer.GetIndex(); actual != "" {
		t.Fatalf("index = %q, want empty", actual)
	}
	if tokenizer.HasNext() {
		t.Fatal("HasNext() = true, want false")
	}
}

func TestPropertyTokenizerShouldParseDelimitedProperty(t *testing.T) {
	tokenizer := NewPropertyTokenizer("person.id")

	if actual := tokenizer.GetIndexedName(); actual != "person" {
		t.Fatalf("indexedName = %q, want %q", actual, "person")
	}
	if actual := tokenizer.GetName(); actual != "person" {
		t.Fatalf("name = %q, want %q", actual, "person")
	}
	if actual := tokenizer.GetChildren(); actual != "id" {
		t.Fatalf("children = %q, want %q", actual, "id")
	}
	if tokenizer.GetIndex() != "" {
		t.Fatalf("index = %q, want empty", tokenizer.GetIndex())
	}
	if !tokenizer.HasNext() {
		t.Fatal("HasNext() = false, want true")
	}
}

func TestPropertyTokenizerShouldParseIndexedProperty(t *testing.T) {
	tokenizer := NewPropertyTokenizer("array[0]")

	if actual := tokenizer.GetIndexedName(); actual != "array[0]" {
		t.Fatalf("indexedName = %q, want %q", actual, "array[0]")
	}
	if actual := tokenizer.GetName(); actual != "array" {
		t.Fatalf("name = %q, want %q", actual, "array")
	}
	if actual := tokenizer.GetIndex(); actual != "0" {
		t.Fatalf("index = %q, want %q", actual, "0")
	}
	if tokenizer.HasNext() {
		t.Fatal("HasNext() = true, want false")
	}
}

func TestPropertyTokenizerShouldTreatTrailingDotAsEmptyChildLikeMyBatis(t *testing.T) {
	tokenizer := NewPropertyTokenizer("person.")

	if actual := tokenizer.GetIndexedName(); actual != "person" {
		t.Fatalf("indexedName = %q, want %q", actual, "person")
	}
	if actual := tokenizer.GetName(); actual != "person" {
		t.Fatalf("name = %q, want %q", actual, "person")
	}
	if actual := tokenizer.GetChildren(); actual != "" {
		t.Fatalf("children = %q, want empty", actual)
	}
	if !tokenizer.HasNext() {
		t.Fatal("HasNext() = false, want true")
	}

	next := tokenizer.Next()
	if next == nil {
		t.Fatal("Next() = nil, want empty child tokenizer")
	}
	if actual := next.GetIndexedName(); actual != "" {
		t.Fatalf("next indexedName = %q, want empty", actual)
	}
	if actual := next.GetName(); actual != "" {
		t.Fatalf("next name = %q, want empty", actual)
	}
	if actual := next.GetChildren(); actual != "" {
		t.Fatalf("next children = %q, want empty", actual)
	}
	if actual := next.GetIndex(); actual != "" {
		t.Fatalf("next index = %q, want empty", actual)
	}
	if next.HasNext() {
		t.Fatal("next HasNext() = true, want false")
	}
	if next.Next() != nil {
		t.Fatal("next Next() != nil, want nil")
	}
}

func TestPropertyTokenizerShouldRejectUnclosedBracketLikeMyBatis(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected NewPropertyTokenizer to panic")
		}

		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("panic = %T, want error", recovered)
		}
		if err.Error() != "Invalid index syntax in property: 'foo[bar'. Missing closing bracket." {
			t.Fatalf("error = %q", err.Error())
		}
	}()

	_ = NewPropertyTokenizer("foo[bar")
}
