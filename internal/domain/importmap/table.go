package importmap

type ReadOptions struct {
	Sheets  []string
	MaxRows int
	Letters bool
}

type SheetInfo struct {
	Name    string
	Headers []string
	Rows    int
}

type Origin struct {
	Sheet string
	Row   int
}

type Table struct {
	Headers []string
	Rows    [][]string
	Origins []Origin
}

func (t Table) Origin(index int) Origin {
	if index < 0 || index >= len(t.Origins) {
		return Origin{Row: index + 2}
	}
	return t.Origins[index]
}
