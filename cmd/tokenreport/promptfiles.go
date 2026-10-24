package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/tuning"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	promptStand    = "replica"
	promptsCommand = "go run ./cmd/tokenreport -prompts "
	agentFile      = "agent-first-round.txt"
	agentInput     = "Which pages of the site carry no entity yet, and what would you propose for them?"
	fileMode       = 0o644
	directoryMode  = 0o755
	shortCommit    = 8
)

func promptPaths() []string {
	return []string{"/peptides/", "/tb-500/", "/product/bpc-157-peptide/"}
}

type promptStep struct {
	key  string
	name string
	said string
	note string
}

func promptSteps() []promptStep {
	return []promptStep{
		{key: stepWriter, name: steps.NameGenerateBody, said: "the body writer"},
		{
			key: stepMeta, name: steps.NameGenerateMeta, said: "the meta writer",
			note: "Draft: a stand-in of the template's headings and a summary from the captured prose; a run passes the draft its writer has just written",
		},
		{
			key: stepJudge, name: steps.NameJudge, said: "the judge",
			note: "Body: a stand-in of the template's target words, cut from the captured prose; a run judges the body its writer has just written",
		},
	}
}

func writePromptFiles(ctx context.Context, dir string, rows []pageRow, commit string) error {
	if err := os.MkdirAll(dir, directoryMode); err != nil {
		return err
	}
	policy := tuning.NewPolicy(settings.Default().NewValues())
	for _, path := range promptPaths() {
		row, err := promptRow(rows, path)
		if err != nil {
			return err
		}
		for _, step := range promptSteps() {
			req, held := row.sent[step.key]
			if !held {
				return fmt.Errorf("%s %s sent no %s request", promptStand, path, step.said)
			}
			lines := []string{
				"Page: " + promptStand + " " + path + " (" + row.Type + ")",
				"Step: " + step.name + ", " + step.said,
				"Role: " + string(req.Meta.Role),
				"Model: " + req.Ref.Model,
				"Reasoning effort: " + string(policy.Effort(req.Meta.Role)),
				"Service tier: " + string(policy.Tier(req.Meta.Role)),
				"Most output tokens: " + strconv.Itoa(req.MaxTokens),
			}
			if step.note != "" {
				lines = append(lines, step.note)
			}
			var out bytes.Buffer
			header(&out, lines, dir, commit)
			section(&out, "SYSTEM MESSAGE", req.System)
			for _, message := range req.Messages {
				section(&out, strings.ToUpper(string(message.Role))+" MESSAGE", message.Text)
			}
			schema, err := json.MarshalIndent(req.Schema, "", "  ")
			if err != nil {
				return err
			}
			section(&out, "STRUCTURED ANSWER SCHEMA (a strict JSON schema the answer must follow)", string(schema))
			if err := os.WriteFile(filepath.Join(dir, promptStand+"-"+slugOf(path)+"-"+step.key+".txt"), out.Bytes(), fileMode); err != nil {
				return err
			}
		}
	}
	return writeAgentRound(ctx, dir, policy, commit)
}

func promptRow(rows []pageRow, path string) (pageRow, error) {
	for i := range rows {
		if rows[i].Stand == promptStand && rows[i].Path == path {
			if rows[i].Skipped != "" {
				return pageRow{}, fmt.Errorf("%s %s is not measured: %s", promptStand, path, rows[i].Skipped)
			}
			return rows[i], nil
		}
	}
	return pageRow{}, fmt.Errorf("the %s capture holds no %s", promptStand, path)
}

func writeAgentRound(ctx context.Context, dir string, policy *tuning.Policy, commit string) error {
	model, err := sitemodeltest.Stand(promptStand)
	if err != nil {
		return err
	}
	brief, err := model.Service().Brief(ctx, model.Site.ID)
	if err != nil {
		return err
	}
	known, err := catalog.New(noOverrides{})
	if err != nil {
		return err
	}
	chat, err := known.Default(domainllm.RoleChat)
	if err != nil {
		return err
	}
	situation := siteContextOf(model)
	situation.Mode = string(domainagent.ModeConfirm)
	developer, err := agentapp.Situation(situation)
	if err != nil {
		return err
	}
	situation.Brief = brief.Text()
	system, err := agentapp.Instructions(situation)
	if err != nil {
		return err
	}
	built := tools.New(tools.Deps{}).Build(tools.Binding{Mode: domainagent.ModeConfirm})

	var out bytes.Buffer
	header(&out, []string{
		"Turn: the first round of an agent turn on the " + promptStand + ", in confirm mode, every tool loaded (agent.toolLoading: eager)",
		"Role: " + string(domainllm.RoleChat),
		"Model: " + chat.Model,
		"Reasoning effort: " + string(policy.Effort(domainllm.RoleChat)),
		"Service tier: " + string(policy.Tier(domainllm.RoleChat)),
	}, dir, commit)
	section(&out, "SYSTEM MESSAGE", system)
	section(&out, "DEVELOPER MESSAGE", developer)
	section(&out, "USER MESSAGE", agentInput)
	out.WriteString("=== TOOLS (" + strconv.Itoa(len(built)) + ", as each is sent: its name, description and parameters) ===\n")
	for i := range built {
		encoded, err := json.MarshalIndent(wireTool{Name: built[i].Def.Name, Description: built[i].Def.Description, Parameters: built[i].Def.Schema}, "", "  ")
		if err != nil {
			return err
		}
		out.Write(encoded)
		out.WriteString("\n\n")
	}
	return os.WriteFile(filepath.Join(dir, agentFile), out.Bytes(), fileMode)
}

func header(out *bytes.Buffer, lines []string, dir, commit string) {
	for _, line := range lines {
		out.WriteString(line + "\n")
	}
	out.WriteString("Commit: " + commit + "\n")
	out.WriteString("Written by: " + promptsCommand + filepath.ToSlash(dir) + "\n\n")
}

func section(out *bytes.Buffer, name, text string) {
	out.WriteString("=== " + name + " ===\n" + strings.TrimRight(text, "\n") + "\n\n")
}

func slugOf(path string) string {
	return strings.Trim(strings.ReplaceAll(path, "/", "-"), "-")
}

func commitOf(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		head, err := os.ReadFile(filepath.Join(dir, ".git", "HEAD"))
		if err == nil {
			return resolved(filepath.Join(dir, ".git"), strings.TrimSpace(string(head)))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository above %s", start)
		}
		dir = parent
	}
}

func resolved(git, head string) (string, error) {
	ref, named := strings.CutPrefix(head, "ref: ")
	if !named {
		return head[:min(len(head), shortCommit)], nil
	}
	if loose, err := os.ReadFile(filepath.Join(git, filepath.FromSlash(ref))); err == nil {
		hash := strings.TrimSpace(string(loose))
		return hash[:min(len(hash), shortCommit)], nil
	}
	packed, err := os.ReadFile(filepath.Join(git, "packed-refs"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(packed), "\n") {
		if hash, name, found := strings.Cut(strings.TrimSpace(line), " "); found && name == ref {
			return hash[:min(len(hash), shortCommit)], nil
		}
	}
	return "", fmt.Errorf("the ref %s is not in the repository", ref)
}
