#!/usr/bin/env bash
# Publish the OpenAPI spec and rendered Markdown docs to odc/emeland/openapi,
# ported from the GitHub openapi-publish.yml workflow. Runs on semver tags
# (vX.Y.Z and vX.Y.Z-rcN) and via a manual job with a TAG.
#
# Auth model (GitLab-native, no stored secret):
#   Uses the pipeline's CI_JOB_TOKEN to clone the openapi repo and push a
#   branch, then opens a merge request against its default branch via git push
#   options. A human (or automerge) reviews and merges, so CI never pushes
#   straight to the shared repo's main and there is no long-lived token.
#
#   Requires (one-time, on odc/emeland/openapi): add odc/emeland/modelsrv to
#   the CI/CD job token inbound allowlist and allow it to push. GitLab >= 17.2.
#
# Behaviour preserved from the GitHub workflow:
#   - render docs with tools/oapidoc and copy the spec in as openapi.yaml
#   - modelsrv/<tag>/ is immutable: identical content is a no-op, different
#     content is a hard error
#   - modelsrv/latest/ is updated only when <tag> is the highest version
#   - both READMEs are regenerated (with GitLab project URLs)
#
# Env (set by the job): OPENAPI_SPEC, OPENAPI_REPO_PATH, OPENAPI_REPO_BRANCH.
set -euo pipefail

TAG="${1:?tag required, e.g. v0.11.0}"
if [[ ! "$TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc[0-9]+)?$ ]]; then
  echo "Refusing tag (want vX.Y.Z or vX.Y.Z-rcN): $TAG" >&2
  exit 1
fi

OPENAPI_SPEC="${OPENAPI_SPEC:?OPENAPI_SPEC is required}"
OPENAPI_REPO_PATH="${OPENAPI_REPO_PATH:-odc/emeland/openapi}"
OPENAPI_REPO_BRANCH="${OPENAPI_REPO_BRANCH:-main}"
: "${CI_JOB_TOKEN:?CI_JOB_TOKEN is required (run inside a GitLab pipeline)}"

# 1. Render the docs for this tag and include the raw spec.
go run ./tools/oapidoc -spec "$OPENAPI_SPEC" -out dist/apidocs -version "$TAG"
cp "$OPENAPI_SPEC" dist/apidocs/openapi.yaml

# 2. Clone the openapi repo with the job token.
repo_host="${CI_SERVER_HOST:-gitlab.opendefense.cloud}"
clone_url="https://gitlab-ci-token:${CI_JOB_TOKEN}@${repo_host}/${OPENAPI_REPO_PATH}.git"
rm -rf openapi-repo
git clone --branch "$OPENAPI_REPO_BRANCH" --depth 1 "$clone_url" openapi-repo

project_url="${CI_SERVER_URL:-https://$repo_host}/odc/emeland/modelsrv"
DEST="openapi-repo/modelsrv/${TAG}"

# 3. Immutable per-tag publish.
if [ -d "$DEST" ]; then
  if diff -rq dist/apidocs "$DEST" >/dev/null; then
    echo "modelsrv/${TAG} already published with identical content; nothing to push."
    exit 0
  fi
  echo "ERROR: modelsrv/${TAG} already exists with different content (releases are immutable)." >&2
  diff -rq dist/apidocs "$DEST" >&2 || true
  exit 1
fi
mkdir -p openapi-repo/modelsrv
cp -a dist/apidocs "$DEST"

# 4. Update latest/ only if this tag is the highest published version.
HIGHEST="$(find openapi-repo/modelsrv -maxdepth 1 -type d -name 'v*' -printf '%f\n' | sort -V | tail -1)"
if [ "$HIGHEST" = "$TAG" ]; then
  rm -rf openapi-repo/modelsrv/latest
  cp -a dist/apidocs openapi-repo/modelsrv/latest
  echo "Updated modelsrv/latest to ${TAG}"
else
  echo "Skipping latest/ update (${TAG} is not highest; highest is ${HIGHEST})"
fi

# 5. Regenerate READMEs (GitLab URLs).
{
  echo "# modelsrv"
  echo
  echo "Published OpenAPI specs and rendered Markdown docs for [modelsrv](${project_url})."
  echo
  echo "- [latest](latest/) — mirror of the newest release"
  echo
  echo "## Versions"
  echo
  find openapi-repo/modelsrv -maxdepth 1 -type d -name 'v*' -printf '%f\n' | sort -Vr | while read -r v; do
    echo "- [${v}](${v}/)"
  done
} > openapi-repo/modelsrv/README.md

{
  echo "# EmELand OpenAPI"
  echo
  echo "Versioned OpenAPI specifications and browsable Markdown docs published from EmELand services."
  echo
  echo "## APIs"
  echo
  echo "- [modelsrv](modelsrv/) — Emerging Enterprise Landscape API"
} > openapi-repo/README.md

# 6. Commit on a deterministic branch and open an MR (no push to main).
cd openapi-repo
git config user.name "modelsrv CI"
git config user.email "ci@emeland.local"
BRANCH="publish/modelsrv-${TAG}"
git checkout -b "$BRANCH"
git add -A
if git diff --cached --quiet; then
  echo "No changes to commit."
  exit 0
fi
git commit -m "modelsrv ${TAG}: publish OpenAPI spec and docs"
# Force-update the deterministic branch so re-runs are idempotent, and ask
# GitLab to open (or update) an MR targeting the default branch.
git push --force "$clone_url" "HEAD:${BRANCH}" \
  -o merge_request.create \
  -o merge_request.target="${OPENAPI_REPO_BRANCH}" \
  -o merge_request.title="modelsrv ${TAG}: publish OpenAPI spec and docs" \
  -o merge_request.remove_source_branch
echo "Opened/updated merge request for ${BRANCH} -> ${OPENAPI_REPO_BRANCH} in ${OPENAPI_REPO_PATH}."
