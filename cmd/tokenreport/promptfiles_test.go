package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestThePromptFilesHoldWhatTheStepsWouldSend(t *testing.T) {
	t.Parallel()

	rows := measuredPages(t)
	first, second := t.TempDir(), t.TempDir()
	for _, dir := range []string{first, second} {
		if err := writePromptFiles(t.Context(), dir, rows, "abc1234"); err != nil {
			t.Fatalf("writePromptFiles: %v", err)
		}
	}

	entries, err := os.ReadDir(first)
	if err != nil {
		t.Fatalf("read %s: %v", first, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	want := []string{
		"agent-first-round.txt",
		"replica-peptides-judge.txt", "replica-peptides-meta.txt", "replica-peptides-writer.txt",
		"replica-product-bpc-157-peptide-judge.txt", "replica-product-bpc-157-peptide-meta.txt", "replica-product-bpc-157-peptide-writer.txt",
		"replica-tb-500-judge.txt", "replica-tb-500-meta.txt", "replica-tb-500-writer.txt",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("files = %v, want %v", names, want)
	}

	for _, name := range names {
		written, readErr := os.ReadFile(filepath.Join(first, name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		again, againErr := os.ReadFile(filepath.Join(second, name))
		if againErr != nil || !bytes.Equal(written, bytes.ReplaceAll(again, []byte(filepath.ToSlash(second)), []byte(filepath.ToSlash(first)))) {
			t.Errorf("%s differs between two writes on one commit, beyond the directory it names", name)
		}
		text := string(written)
		for _, said := range []string{"Model: gpt-", "Commit: abc1234", "go run ./cmd/tokenreport -prompts", "=== SYSTEM MESSAGE ===", "=== USER MESSAGE ==="} {
			if !strings.Contains(text, said) {
				t.Errorf("%s never says %q", name, said)
			}
		}
		if name == agentFile {
			continue
		}
		for _, said := range []string{"Page: replica /", "Step: ", "=== STRUCTURED ANSWER SCHEMA"} {
			if !strings.Contains(text, said) {
				t.Errorf("%s never says %q", name, said)
			}
		}
		for _, placeholder := range template.Placeholders() {
			if strings.Contains(text, placeholder) {
				t.Errorf("%s carries the placeholder %s unexpanded", name, placeholder)
			}
		}
	}
	for _, name := range names {
		if filepath.Ext(name) != ".txt" {
			t.Errorf("%s is no plain text file", name)
		}
	}

	agent, err := os.ReadFile(filepath.Join(first, "agent-first-round.txt"))
	if err != nil {
		t.Fatalf("read the agent's round: %v", err)
	}
	for _, said := range []string{"=== DEVELOPER MESSAGE ===", "=== TOOLS (", "graph_propose", "findings"} {
		if !strings.Contains(string(agent), said) {
			t.Errorf("the agent's round never says %q", said)
		}
	}
}
