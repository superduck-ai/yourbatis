package parsing

import (
	"fmt"
	"strings"
)

type PropertyTokenizer struct {
	name        string
	indexedName string
	index       string
	children    string
	hasChildren bool
}

func NewPropertyTokenizer(fullname string) *PropertyTokenizer {
	tokenizer := &PropertyTokenizer{}
	delim := strings.Index(fullname, ".")
	if delim > -1 {
		tokenizer.name = fullname[:delim]
		tokenizer.children = fullname[delim+1:]
		tokenizer.hasChildren = true
	} else {
		tokenizer.name = fullname
	}

	tokenizer.indexedName = tokenizer.name
	delim = strings.Index(tokenizer.name, "[")
	if delim > -1 {
		if !tokenizer.hasChildren && !strings.HasSuffix(tokenizer.name, "]") {
			panic(fmt.Errorf("Invalid index syntax in property: '%s'. Missing closing bracket.", tokenizer.name))
		}
		tokenizer.index = tokenizer.name[delim+1 : len(tokenizer.name)-1]
		tokenizer.name = tokenizer.name[:delim]
	}

	return tokenizer
}

func (tokenizer *PropertyTokenizer) GetName() string {
	if tokenizer == nil {
		return ""
	}
	return tokenizer.name
}

func (tokenizer *PropertyTokenizer) GetIndex() string {
	if tokenizer == nil {
		return ""
	}
	return tokenizer.index
}

func (tokenizer *PropertyTokenizer) GetIndexedName() string {
	if tokenizer == nil {
		return ""
	}
	return tokenizer.indexedName
}

func (tokenizer *PropertyTokenizer) GetChildren() string {
	if tokenizer == nil {
		return ""
	}
	return tokenizer.children
}

func (tokenizer *PropertyTokenizer) HasNext() bool {
	return tokenizer != nil && tokenizer.hasChildren
}

func (tokenizer *PropertyTokenizer) Next() *PropertyTokenizer {
	if tokenizer == nil || !tokenizer.hasChildren {
		return nil
	}
	return NewPropertyTokenizer(tokenizer.children)
}
