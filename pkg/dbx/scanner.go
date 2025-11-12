package dbx

type RowScanner interface {
	Scan(dest ...any) error
}
