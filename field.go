package korm

type Field struct {
	TableName string
	Columns   []string
	ColumnMap map[string]string
}

func newField(tableName string) *Field {
	return &Field{
		TableName: tableName,
		ColumnMap: make(map[string]string),
	}
}

func (f *Field) addColumns(column []string) {
	f.Columns = append(f.Columns, column...)
	for _, c := range column {
		f.ColumnMap[c] = c
	}
}
