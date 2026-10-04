package run

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"
)

type ArtifactKind string

const (
	ArtifactLinkContext      ArtifactKind = "link_context"
	ArtifactDraft            ArtifactKind = "draft"
	ArtifactBodyHTML         ArtifactKind = "body_html"
	ArtifactMeta             ArtifactKind = "meta"
	ArtifactImages           ArtifactKind = "images"
	ArtifactValidationReport ArtifactKind = "validation_report"
	ArtifactJudgeReport      ArtifactKind = "judge_report"
	ArtifactPublishResult    ArtifactKind = "publish_result"
	ArtifactRelinkResult     ArtifactKind = "relink_result"
	ArtifactSyncResult       ArtifactKind = "sync_result"
	ArtifactFinalReport      ArtifactKind = "final_report"
	ArtifactRevertResult     ArtifactKind = "revert_result"
)

func ArtifactKinds() []ArtifactKind {
	return []ArtifactKind{
		ArtifactLinkContext, ArtifactDraft, ArtifactBodyHTML, ArtifactMeta, ArtifactImages,
		ArtifactValidationReport, ArtifactJudgeReport, ArtifactPublishResult, ArtifactRelinkResult,
		ArtifactSyncResult, ArtifactFinalReport, ArtifactRevertResult,
	}
}

func (k ArtifactKind) Valid() bool {
	return slices.Contains(ArtifactKinds(), k)
}

type Artifact struct {
	ID        string
	RunID     string
	ItemID    string
	Step      string
	Kind      ArtifactKind
	Blob      []byte
	Size      int
	Hash      string
	Purged    bool
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func HashBlob(blob []byte) string {
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

func NewArtifact(a Artifact) (Artifact, error) {
	switch {
	case a.ID == "":
		return Artifact{}, invalid("artifact id must not be empty", "id")
	case a.RunID == "":
		return Artifact{}, invalid("artifact run id must not be empty", "runId")
	case a.ItemID == "":
		return Artifact{}, invalid("artifact item id must not be empty", "itemId")
	case a.Step == "":
		return Artifact{}, invalid("artifact step must not be empty", "step")
	case !a.Kind.Valid():
		return Artifact{}, invalid("artifact kind is not recognized", "kind")
	}

	a.Size = len(a.Blob)
	a.Hash = HashBlob(a.Blob)
	return a, nil
}
