package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var corePrefixes = []string{"internal/domain/", "internal/application/"}

type block struct {
	file       string
	statements int
	covered    bool
}

type profile struct {
	blocks []block
}

type tally struct {
	statements int
	covered    int
}

func (t tally) percent() float64 {
	if t.statements == 0 {
		return 0
	}
	return float64(t.covered) / float64(t.statements) * 100
}

type gates struct {
	core  float64
	total float64
}

type result struct {
	name      string
	tally     tally
	threshold float64
	skipped   bool
}

func (r result) met() bool {
	return r.skipped || r.tally.percent() >= r.threshold
}

func (r result) String() string {
	if r.skipped {
		return fmt.Sprintf("%-20s no statements yet, gate of %.1f%% skipped", r.name, r.threshold)
	}

	verdict := "FAIL"
	if r.met() {
		verdict = "ok"
	}
	return fmt.Sprintf("%-20s %6.2f%% of %d statements, gate %.1f%% %s", r.name, r.tally.percent(), r.tally.statements, r.threshold, verdict)
}

type report struct {
	core  result
	total result
}

func (r report) pass() bool {
	return r.core.met() && r.total.met()
}

func (r report) String() string {
	return r.core.String() + "\n" + r.total.String()
}

func parse(source io.Reader) (profile, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return profile{}, fmt.Errorf("read coverage profile: %w", err)
		}
		return profile{}, fmt.Errorf("coverage profile is empty")
	}
	if !strings.HasPrefix(scanner.Text(), "mode:") {
		return profile{}, fmt.Errorf("coverage profile does not start with a mode line")
	}

	var parsed profile
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry, err := parseBlock(line)
		if err != nil {
			return profile{}, err
		}
		parsed.blocks = append(parsed.blocks, entry)
	}
	if err := scanner.Err(); err != nil {
		return profile{}, fmt.Errorf("read coverage profile: %w", err)
	}
	return parsed, nil
}

func parseBlock(line string) (block, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return block{}, fmt.Errorf("coverage line %q does not have three fields", line)
	}

	file, _, found := strings.Cut(fields[0], ":")
	if !found || file == "" {
		return block{}, fmt.Errorf("coverage line %q does not name a file", line)
	}

	statements, err := strconv.Atoi(fields[1])
	if err != nil {
		return block{}, fmt.Errorf("coverage line %q has an unreadable statement count: %w", line, err)
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return block{}, fmt.Errorf("coverage line %q has an unreadable hit count: %w", line, err)
	}

	return block{file: file, statements: statements, covered: count > 0}, nil
}

func (p profile) rate(prefixes []string) tally {
	var counted tally
	for _, entry := range p.blocks {
		if !matches(entry.file, prefixes) {
			continue
		}
		counted.statements += entry.statements
		if entry.covered {
			counted.covered += entry.statements
		}
	}
	return counted
}

func matches(file string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		if strings.Contains(file, prefix) {
			return true
		}
	}
	return false
}

func evaluate(p profile, thresholds gates) report {
	core := p.rate(corePrefixes)
	total := p.rate(nil)

	return report{
		core:  result{name: "domain+application", tally: core, threshold: thresholds.core, skipped: core.statements == 0},
		total: result{name: "total", tally: total, threshold: thresholds.total, skipped: total.statements == 0},
	}
}
