package artifact

import "github.com/google/uuid"

// ArtifactRef references an [Artifact] by resolved object and/or id.
type ArtifactRef struct {
	Artifact   Artifact
	ArtifactId uuid.UUID
}

// ResolvedArtifact returns the embedded [Artifact] when present, or nil.
func (r *ArtifactRef) ResolvedArtifact() Artifact {
	if r == nil {
		return nil
	}
	return r.Artifact
}
