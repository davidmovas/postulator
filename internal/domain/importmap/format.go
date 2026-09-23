package importmap

type Format string

const (
	FormatXLSX Format = "xlsx"
	FormatCSV  Format = "csv"
)

func (f Format) Valid() bool {
	switch f {
	case FormatXLSX, FormatCSV:
		return true
	default:
		return false
	}
}

func (f Format) Extension() string {
	return "." + string(f)
}
