package run

import "slices"

type RetryBlockedReason string

const RetryBlockedInputsExpired RetryBlockedReason = "inputs_expired"

var purgeableArtifactKinds = []ArtifactKind{ArtifactDraft, ArtifactBodyHTML, ArtifactImages}

func PurgeableArtifactKinds() []ArtifactKind {
	return slices.Clone(purgeableArtifactKinds)
}

func (k ArtifactKind) Purgeable() bool {
	return slices.Contains(purgeableArtifactKinds, k)
}

func PurgedKinds(artifacts []Artifact) []ArtifactKind {
	kinds := make([]ArtifactKind, 0, len(artifacts))
	for i := range artifacts {
		if artifacts[i].Purged {
			kinds = append(kinds, artifacts[i].Kind)
		}
	}
	return kinds
}

func ExpiredInputs(requires, purged []ArtifactKind) []ArtifactKind {
	expired := make([]ArtifactKind, 0, len(requires))
	for _, kind := range requires {
		if slices.Contains(purged, kind) {
			expired = append(expired, kind)
		}
	}
	return expired
}

func RetryBlocked(requires, purged []ArtifactKind) RetryBlockedReason {
	if len(ExpiredInputs(requires, purged)) > 0 {
		return RetryBlockedInputsExpired
	}
	return ""
}
