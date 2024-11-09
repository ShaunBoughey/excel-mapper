package config

type FieldConfig struct {
	Fields          []Field  `json:"fields"`
	MandatoryFields []string `json:"mandatoryFields"`
}

type Field struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	IsMandatory bool   `json:"isMandatory"`
}

func (fc *FieldConfig) GetOrderedFields() []string {
	order := make([]string, len(fc.Fields))
	for i, field := range fc.Fields {
		order[i] = field.Name
	}
	return order
}

func (fc *FieldConfig) GetDisplayNames() map[string]string {
	displayNames := make(map[string]string)
	for _, field := range fc.Fields {
		displayNames[field.Name] = field.DisplayName
	}
	return displayNames
}

func (fc *FieldConfig) GetMandatoryFields() []string {
	var mandatory []string
	for _, field := range fc.Fields {
		if field.IsMandatory {
			mandatory = append(mandatory, field.DisplayName)
		}
	}
	return mandatory
}
