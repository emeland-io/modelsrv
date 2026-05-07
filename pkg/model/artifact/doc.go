// Package artifact provides the Artifact and ArtifactInstance domain types
// for phase 8 of the EmELand model (data catalog / binary artefact tracking).
//
// # Well-known Annotations
//
// ArtifactInstance resources use the following annotation keys:
//
//   - emeland.io/p8-artifact-instance-location — JSON list of URL(s) where a copy of the artifact can be found.
//   - emeland.io/p8-artifact-instance-credentials-ref — reference information on where to find any required credentials for accessing the artifact copy.
package artifact
