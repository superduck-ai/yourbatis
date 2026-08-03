package parsing

type Properties map[string]string

func NewProperties() Properties {
	return make(Properties)
}

func (properties Properties) ContainsKey(key string) bool {
	if properties == nil {
		return false
	}
	_, ok := properties[key]
	return ok
}

func (properties Properties) GetProperty(key string, defaultValue ...string) string {
	if properties != nil {
		if value, ok := properties[key]; ok {
			return value
		}
	}
	if len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return ""
}

func (properties Properties) Put(key, value string) {
	if properties == nil {
		return
	}
	properties[key] = value
}

func (properties Properties) PutAll(other Properties) {
	if properties == nil {
		return
	}
	for key, value := range other {
		properties[key] = value
	}
}

func (properties Properties) Clone() Properties {
	if properties == nil {
		return nil
	}
	clone := make(Properties, len(properties))
	for key, value := range properties {
		clone[key] = value
	}
	return clone
}
