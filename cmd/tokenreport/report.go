package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

type report struct {
	Models []modelRow  `json:"models"`
	Agent  agentReport `json:"agent"`
	Pages  []pageRow   `json:"pages"`
}

type modelRow struct {
	Model    string `json:"model"`
	Encoding string `json:"encoding"`
	Fallback bool   `json:"fallback"`
}

func build(ctx context.Context) (report, error) {
	known, err := catalog.New(noOverrides{})
	if err != nil {
		return report{}, err
	}
	mapped, err := catalogEncodings(ctx, known)
	if err != nil {
		return report{}, err
	}
	models := make([]modelRow, 0, len(mapped))
	for _, row := range mapped {
		models = append(models, modelRow{Model: row.model, Encoding: string(row.encoding), Fallback: row.fallback})
	}

	held := newCounters()
	chat, err := known.Default(llm.RoleChat)
	if err != nil {
		return report{}, err
	}
	c, err := held.of(chat)
	if err != nil {
		return report{}, err
	}
	agent, err := measureAgent(ctx, c)
	if err != nil {
		return report{}, err
	}
	pages, err := measurePages(ctx, held, known)
	if err != nil {
		return report{}, err
	}
	return report{Models: models, Agent: agent, Pages: pages}, nil
}

func (r report) write(out io.Writer) error {
	err := tables(out, func(s *sheet) {
		s.heading("MODEL", "ENCODING", "")
		for _, row := range r.Models {
			said := ""
			if row.Fallback {
				said = "not mapped, counted as " + row.Encoding
			}
			s.row(row.Model, row.Encoding, said)
		}
	})
	if err != nil {
		return err
	}
	if err := r.Agent.write(out); err != nil {
		return err
	}
	return writePages(out, r.Pages)
}

func tables(out io.Writer, fills ...func(*sheet)) error {
	for _, fill := range fills {
		s := newSheet(out)
		fill(s)
		if err := s.flush(); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
	}
	return nil
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

func (s *sheet) measured(name string, m measure) {
	s.row(name, fmt.Sprintf("%7d", m.Bytes), fmt.Sprintf("%7d", m.Tokens))
}

func (s *sheet) flush() error {
	if s.err != nil {
		return s.err
	}
	return s.table.Flush()
}
