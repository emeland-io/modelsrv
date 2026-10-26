#!/usr/bin/env bash
# Upsert a marked container-image block on the GitHub Release for RELEASE_TAG.
#
# Required env:
#   RELEASE_TAG  — git tag / release name (e.g. v0.13.0)
#   IMAGE        — registry image without tag (e.g. ghcr.io/emeland-io/modelsrv)
#   DIGEST       — digest from docker/build-push-action (e.g. sha256:…)
#   MARKER       — block id used in HTML comments (e.g. modelsrv or emelandctl)
#   GITHUB_REPOSITORY — owner/repo (set automatically in Actions)
set -euo pipefail

: "${RELEASE_TAG:?RELEASE_TAG is required}"
: "${IMAGE:?IMAGE is required}"
: "${DIGEST:?DIGEST is required}"
: "${MARKER:?MARKER is required}"
: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY is required}"

START="<!-- container-image:${MARKER} -->"
END="<!-- /container-image:${MARKER} -->"
PACKAGE_URL="https://github.com/${GITHUB_REPOSITORY}/pkgs/container/${MARKER}"

ensure_release() {
  if gh release view "${RELEASE_TAG}" >/dev/null 2>&1; then
    return 0
  fi
  local -a create_args=(release create "${RELEASE_TAG}" --generate-notes)
  # Pre-release tags include a hyphen (e.g. v0.9.3-rc1).
  if [[ "${RELEASE_TAG}" == *-* ]]; then
    create_args+=(--prerelease)
  fi
  gh "${create_args[@]}"
}

build_block() {
  cat <<EOF
${START}
### \`${MARKER}\`

\`\`\`bash
docker pull ${IMAGE}:${RELEASE_TAG}
docker pull ${IMAGE}@${DIGEST}
\`\`\`

**Digest:** \`${DIGEST}\`

**Package:** [${MARKER}](${PACKAGE_URL})
${END}
EOF
}

upsert_body() {
  local body block
  body="$(gh release view "${RELEASE_TAG}" --json body --jq '.body // ""')"
  block="$(build_block)"

  # Replace an existing marked block, or append under ## Container images.
  body="$(
    BODY="${body}" BLOCK="${block}" START="${START}" END="${END}" python3 - <<'PY'
import os
import re

body = os.environ["BODY"]
block = os.environ["BLOCK"].rstrip("\n")
start = os.environ["START"]
end = os.environ["END"]

pattern = re.compile(
    re.escape(start) + r".*?" + re.escape(end),
    re.DOTALL,
)
if pattern.search(body):
    body = pattern.sub(block, body, count=1)
else:
    heading = "## Container images"
    if heading not in body:
        if body and not body.endswith("\n"):
            body += "\n"
        if body and not body.endswith("\n\n"):
            body += "\n"
        body += heading + "\n"
    if body and not body.endswith("\n"):
        body += "\n"
    body += "\n" + block + "\n"

print(body, end="" if body.endswith("\n") else "\n")
PY
  )"

  local notes_file
  notes_file="$(mktemp)"
  printf '%s' "${body}" >"${notes_file}"
  gh release edit "${RELEASE_TAG}" --notes-file "${notes_file}"
  rm -f "${notes_file}"
}

ensure_release
upsert_body
echo "Updated release ${RELEASE_TAG} with image ${IMAGE}@${DIGEST}"
