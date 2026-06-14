package route

type FieldType string

const (
	String FieldType = "string"
	Int    FieldType = "int"
	Bool   FieldType = "bool"
)

type Field struct {
	Name     string
	Type     FieldType
	Required bool
}

type ActionSchema struct {
	Action string
	Fields []Field
}

type Schema struct {
	Actions map[string]ActionSchema
}
