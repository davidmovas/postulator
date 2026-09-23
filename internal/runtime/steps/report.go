package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	NameReport = string(run.StepReport)

	CodeArtifactPurged = "artifact_purged"
)

type FinalReport struct {
	Validation *ValidationReport `json:"validation,omitempty"`
	Judge      *JudgeReport      `json:"judge,omitempty"`
	Publish    *PublishResult    `json:"publish,omitempty"`
	Relink     *RelinkResult     `json:"relink,omitempty"`
	Sync       *SyncResult       `json:"sync,omitempty"`
	Images     *ImagesResult     `json:"images,omitempty"`
	Score      *float64          `json:"score,omitempty"`
	PageID     string            `json:"pageId"`
	Path       string            `json:"path"`
	Findings   []content.Finding `json:"findings"`
	Errors     int               `json:"errors"`
	Warnings   int               `json:"warnings"`
}

func Report(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameReport,
		Produces: []run.ArtifactKind{run.ArtifactFinalReport},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  pureStepTimeout,
		Run: func(_ context.Context, sc *run.StepContext) (run.Result, error) {
			report := FinalReport{
				PageID:   sc.Page.ID,
				Path:     sc.Page.Path,
				Findings: purgedFindings(sc),
			}

			validation, found, err := decodeArtifact[ValidationReport](sc, run.ArtifactValidationReport)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Validation = &validation
				report.Score = &validation.Score
				report.Findings = append(report.Findings, validation.Compliance.Items...)
				report.Findings = append(report.Findings, validation.Structure.Items...)
			}

			judged, found, err := decodeArtifact[JudgeReport](sc, run.ArtifactJudgeReport)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Judge = &judged
				report.Score = lower(report.Score, judged.Score)
				report.Findings = append(report.Findings, judged.Findings...)
			}

			published, found, err := decodeArtifact[PublishResult](sc, run.ArtifactPublishResult)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Publish = &published
				report.Findings = append(report.Findings, published.Findings...)
			}

			relinked, found, err := decodeArtifact[RelinkResult](sc, run.ArtifactRelinkResult)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Relink = &relinked
				report.Findings = append(report.Findings, relinked.Findings...)
			}

			synced, found, err := decodeArtifact[SyncResult](sc, run.ArtifactSyncResult)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Sync = &synced
				report.Findings = append(report.Findings, synced.Findings...)
			}

			pictures, found, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Images = &pictures
				report.Findings = append(report.Findings, pictures.Findings...)
			}

			snippet, found, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Findings = append(report.Findings, snippet.Findings...)
			}

			report.Errors, report.Warnings = weigh(report.Findings)

			blob, err := encode(report, "final report")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactFinalReport, Blob: blob}},
				Message:   scoreline(report),
			}, nil
		},
	}
}

var reportSections = []run.ArtifactKind{
	run.ArtifactValidationReport, run.ArtifactJudgeReport, run.ArtifactPublishResult,
	run.ArtifactRelinkResult, run.ArtifactSyncResult, run.ArtifactImages,
}

func purgedFindings(sc *run.StepContext) []content.Finding {
	out := make([]content.Finding, 0)
	for _, kind := range reportSections {
		artifact, held := sc.Artifacts[kind]
		if !held || !artifact.Purged {
			continue
		}
		out = append(out, content.Finding{
			Severity: content.SeverityWarn,
			Code:     CodeArtifactPurged,
			Message: "the " + string(kind) + " of " + sc.Page.Path +
				" was dropped by retention before the report was assembled, so this report does not carry it",
			Details: map[string]any{"pageId": sc.Page.ID, "path": sc.Page.Path, "kind": string(kind)},
		})
	}
	return out
}

func lower(running, candidate *float64) *float64 {
	switch {
	case candidate == nil:
		return running
	case running == nil:
		return candidate
	case *candidate < *running:
		return candidate
	default:
		return running
	}
}

func scoreline(report FinalReport) string {
	scored := "is not scored"
	if report.Score != nil {
		scored = "scored " + strconv.FormatFloat(*report.Score, 'f', 2, 64)
	}
	return report.Path + " " + scored + " with " + strconv.Itoa(report.Errors) + " errors"
}

func weigh(findings []content.Finding) (errorCount, warnCount int) {
	for i := range findings {
		switch findings[i].Severity {
		case content.SeverityError:
			errorCount++
		case content.SeverityWarn:
			warnCount++
		case content.SeverityInfo:
		}
	}
	return errorCount, warnCount
}
