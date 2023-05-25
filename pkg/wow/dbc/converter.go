package dbc

type DBCDefinition[C any] struct {
	FileName string
	Format   RecordFormat
}

var definitions = []DBCDefinition[any]{{
	"", []FieldType(""),
}}
