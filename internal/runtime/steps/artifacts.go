package steps

import (
	"encoding/json"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func encode(value any, what string) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the "+what)
	}
	return encoded, nil
}

func linkContextOf(sc *run.StepContext) (content.LinkContext, error) {
	artifact, err := sc.Artifact(run.ArtifactLinkContext)
	if err != nil {
		return content.LinkContext{}, err
	}

	var lc content.LinkContext
	if unmarshalErr := json.Unmarshal(artifact.Blob, &lc); unmarshalErr != nil {
		return content.LinkContext{}, errors.Wrap(unmarshalErr, errors.Internal, "the stored link context is not readable")
	}
	return lc, nil
}

func bodyOf(sc *run.StepContext) (*content.Document, error) {
	artifact, err := sc.Artifact(run.ArtifactBodyHTML)
	if err != nil {
		return nil, err
	}
	return content.Parse(string(artifact.Blob))
}

func draftOf(sc *run.StepContext) (content.ContentDraft, error) {
	artifact, err := sc.Artifact(run.ArtifactDraft)
	if err != nil {
		return content.ContentDraft{}, err
	}

	var decoded content.ContentDraft
	if unmarshalErr := json.Unmarshal(artifact.Blob, &decoded); unmarshalErr != nil {
		return content.ContentDraft{}, errors.Wrap(unmarshalErr, errors.Internal, "the stored draft is not readable")
	}
	return decoded, nil
}

func blobOf[T any](stored []run.Artifact, kind run.ArtifactKind) (value T, found bool, err error) {
	for i := range stored {
		if stored[i].Kind != kind || stored[i].Purged || len(stored[i].Blob) == 0 {
			continue
		}
		if unmarshalErr := json.Unmarshal(stored[i].Blob, &value); unmarshalErr != nil {
			return value, false, errors.Wrap(unmarshalErr, errors.Internal,
				"the stored "+string(kind)+" is not readable")
		}
		return value, true, nil
	}
	return value, false, nil
}

func decodeArtifact[T any](sc *run.StepContext, kind run.ArtifactKind) (value T, found bool, err error) {
	artifact, ok := sc.Artifacts[kind]
	if !ok || artifact.Purged || len(artifact.Blob) == 0 {
		return value, false, nil
	}
	if unmarshalErr := json.Unmarshal(artifact.Blob, &value); unmarshalErr != nil {
		return value, false, errors.Wrap(unmarshalErr, errors.Internal, "the stored "+string(kind)+" is not readable")
	}
	return value, true, nil
}
