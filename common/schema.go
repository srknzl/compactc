package common

type Schema struct {
	Import    []string `yaml,json:"import"`    // optional
	Namespace string   `yaml,json:"namespace"` // optional, by default DefaultNamespace
	Classes   []Class  `yaml,json:"classes"`
}

type Class struct {
	Name      string  `yaml,json:"name"`
	Fields    []Field `yaml,json:"fields"`
	Namespace string
}

type Field struct {
	Name     string `yaml,json:"name"`
	Type     string `yaml,json:"type"`
	External bool   `yaml,json:"external"` // optional
}

type ClassAndFileName struct {
	FileName  string
	ClassName string
}
