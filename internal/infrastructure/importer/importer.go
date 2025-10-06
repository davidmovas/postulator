package importer

type Importer interface {
	Import(filePath string) ([]string, error)
}
