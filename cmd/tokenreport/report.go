package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/tiktoken-go/tokenizer"
)

type report struct {
	Models []modelRow  `json:"models"`
	Agent  agentReport `json:"agent"`
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

	c, err := newCounter(tokenizer.O200kBase)
	if err != nil {
		return report{}, err
	}
	agent, err := measureAgent(ctx, c)
	if err != nil {
		return report{}, err
	}
	return report{Models: models, Agent: agent}, nil
}

func (r report) write(out io.Writer) error {
	s := newSheet(out)
	s.heading("MODEL", "ENCODING", "")
	for _, row := range r.Models {
		said := ""
		if row.Fallback {
			said = "not mapped, counted as " + row.Encoding
		}
		s.row(row.Model, row.Encoding, said)
	}
	s.blank()
	if err := s.flush(); err != nil {
		return err
	}
	return r.Agent.write(out)
}

type sheet struct {
	table *tabwriter.Writer
	err   error
}

func newSheet(out io.Writer) *sheet {
	return &sheet{table: tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)}
}

func (s *sheet) row(cells ...string) {
	if s.err != nil {
		return
	}
	_, s.err = fmt.Fprintln(s.table, strings.Join(cells, "\t")+"\t")
}

func (s *sheet) heading(cells ...string) {
	s.row(cells...)
}

func (s *sheet) blank() {
	s.row("")
}

func (s *sheet) measured(name string, m measure) {
	s.row(name, fmt.Sprintf("%7d", m.Bytes), fmt.Sprintf("%7d", m.Tokens))
}

func (s *sheet) flush() error {
	if s.err != nil {
		return s.err
	}
	return s.table.Flush()
}
