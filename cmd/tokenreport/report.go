package main

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
)

type report struct {
	Models []modelRow `json:"models"`
}

type modelRow struct {
	Model    string `json:"model"`
	Encoding string `json:"encoding"`
	Fallback bool   `json:"fallback"`
}

func build(ctx context.Context) (report, error) {
	mapped, err := catalogEncodings(ctx)
	if err != nil {
		return report{}, err
	}
	models := make([]modelRow, 0, len(mapped))
	for _, row := range mapped {
		models = append(models, modelRow{Model: row.model, Encoding: string(row.encoding), Fallback: row.fallback})
	}
	return report{Models: models}, nil
}

func (r report) write(out io.Writer) error {
	table := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "MODEL\tENCODING\t"); err != nil {
		return err
	}
	for _, row := range r.Models {
		said := ""
		if row.Fallback {
			said = "not mapped, counted as " + row.Encoding
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\n", row.Model, row.Encoding, said); err != nil {
			return err
		}
	}
	return table.Flush()
}
