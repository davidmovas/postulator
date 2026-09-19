package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const NameReport = string(run.StepReport)

type FinalReport struct {
	Validation *ValidationReport `json:"validation,omitempty"`
	Judge      *JudgeReport      `json:"judge,omitempty"`
	Publish    *PublishResult    `json:"publish,omitempty"`
	Relink     *RelinkResult     `json:"relink,omitempty"`
	Sync       *SyncResult       `json:"sync,omitempty"`
	Images     *ImagesResult     `json:"images,omitempty"`
	PageID     string            `json:"pageId"`
	Path       string            `json:"path"`
	Findings   []content.Finding `json:"findings"`
	Score      float64           `json:"score"`
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
				Findings: make([]content.Finding, 0),
				Score:    1,
			}

			validation, found, err := decodeArtifact[ValidationReport](sc, run.ArtifactValidationReport)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Validation = &validation
				report.Score = validation.Score
				report.Findings = append(report.Findings, validation.Compliance.Items...)
				report.Findings = append(report.Findings, validation.Structure.Items...)
			}

			judged, found, err := decodeArtifact[JudgeReport](sc, run.ArtifactJudgeReport)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Judge = &judged
				report.Score = min(report.Score, judged.Score)
			}

			published, found, err := decodeArtifact[PublishResult](sc, run.ArtifactPublishResult)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Publish = &published
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
			}

			pictures, found, err := decodeArtifact[ImagesResult](sc, run.ArtifactImages)
			if err != nil {
				return run.Result{}, err
			}
			if found {
				report.Images = &pictures
			}

			report.Errors, report.Warnings = weigh(report.Findings)

			blob, err := encode(report, "final report")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactFinalReport, Blob: blob}},
				Message: sc.Page.Path + " scored " + strconv.FormatFloat(report.Score, 'f', 2, 64) +
					" with " + strconv.Itoa(report.Errors) + " errors",
			}, nil
		},
	}
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
