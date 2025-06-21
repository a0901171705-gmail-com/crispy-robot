#!/usr/bin/env bash
#
# bump-go.sh — Update go.mod `go` directive and toolchain to latest stable Go release.
#
# Usage:
#   ./bump-go.sh [--apply|-a] <path/to/go.mod>
#
# By default the script runs in *dry‑run* mode: it creates a local branch,
# commits the version bump, shows the exact patch, **checks for an existing PR**
# with the same title, and exits.  Nothing is pushed.  The temporary branch is
# deleted automatically on exit, so your working tree stays pristine.  Pass
# --apply (or -a) to push the branch and open a new PR *only if one doesn’t
# already exist*.
# -----------------------------------------------------------------------------
set -euo pipefail

usage() {
  echo "Usage: $0 [--apply|-a] <path/to/go.mod>" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Error: '$1' is required but was not found in PATH" >&2
    exit 1
  }
}

# ---- Preconditions ----------------------------------------------------------
for cmd in curl jq git gh; do
  require_cmd "$cmd"
done

# ---- Argument parsing -------------------------------------------------------
APPLY=0
GO_MOD=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --apply|-a) APPLY=1 ;;
    -h|--help)  usage ;;
    *)          [[ -z "$GO_MOD" ]] && GO_MOD="$1" || usage ;;
  esac
  shift
done

[[ -z "$GO_MOD" ]] && usage
[[ -f "$GO_MOD" ]] || { echo "Error: '$GO_MOD' not found" >&2; exit 1; }

# ---- Discover latest stable Go release --------------------------------------

echo "Fetching latest stable Go version…"
LATEST_JSON=$(curl -fsSL https://go.dev/dl/?mode=json | jq -c '[.[] | select(.stable==true)][0]')
FULL_VERSION=$(jq -r '.version' <<< "$LATEST_JSON")        # go1.23.4
TOOLCHAIN_VERSION="${FULL_VERSION#go}"                      # 1.23.4
GO_DIRECTIVE_VERSION=$(cut -d. -f1-2 <<< "$TOOLCHAIN_VERSION")

echo "  → toolchain : $TOOLCHAIN_VERSION"
echo "  → directive : $GO_DIRECTIVE_VERSION"

# ---- Prepare Git worktree ---------------------------------------------------
BRANCH="bump-go-$TOOLCHAIN_VERSION"
cleanup() {
  git checkout - >/dev/null 2>&1 || true
  git branch -D "$BRANCH" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "Creating branch $BRANCH"
git switch -c "$BRANCH" >/dev/null 2>&1

# ---- Patch go.mod -----------------------------------------------------------
CURRENT_GO_DIRECTIVE=$(grep -E '^go ' "$GO_MOD" | awk '{print $2}')

if [[ "$CURRENT_GO_DIRECTIVE" != "$GO_DIRECTIVE_VERSION" ]]; then
  sed -Ei.bak "s/^go [0-9]+\.[0-9]+.*$/go $GO_DIRECTIVE_VERSION/" "$GO_MOD"
  echo "  • go directive $CURRENT_GO_DIRECTIVE → $GO_DIRECTIVE_VERSION"
fi

if grep -qE '^toolchain' "$GO_MOD"; then
  sed -Ei.bak "s/^toolchain go[0-9]+\.[0-9]+\.[0-9]+.*$/toolchain go$TOOLCHAIN_VERSION/" "$GO_MOD"
  echo "  • updated existing toolchain → go$TOOLCHAIN_VERSION"
else
  printf '\n# updated automatically\ntoolchain go%s\n' "$TOOLCHAIN_VERSION" >> "$GO_MOD"
  echo "  • added toolchain go$TOOLCHAIN_VERSION"
fi

rm -f "$GO_MOD.bak"

git add "$GO_MOD"

# Bail out if nothing changed (after clean‑up via trap)
if git diff --cached --quiet; then
  echo "No version bump required — already on latest Go."
  exit 0
fi

# ---- Commit -----------------------------------------------------------------
COMMIT_MSG="Bump Go to $TOOLCHAIN_VERSION"
git commit -m "$COMMIT_MSG" >/dev/null
COMMIT_HASH=$(git rev-parse --short HEAD)

PR_TITLE="$COMMIT_MSG"

# ---- Check for existing PR ------------------------------------------


existing_pr=$(gh search prs --repo cli/cli --match title "$PR_TITLE" --json title --jq "map(select(.title == \"$PR_TITLE\") | .title) | length > 0")

if [[ "$existing_pr" == "true" ]]; then
  echo "Found an existing open PR titled '$PR_TITLE'. Skipping push/PR creation."
  if [[ $APPLY -eq 0 ]]; then
    echo -e "\n=== DRY‑RUN DIFF (commit $COMMIT_HASH):\n"
    git --no-pager show --color "$COMMIT_HASH"
  fi
  exit 0
fi

# ---- Dry‑run handling -------------------------------------------------------
if [[ $APPLY -eq 0 ]]; then
  echo -e "\n=== DRY‑RUN DIFF (commit $COMMIT_HASH):\n"
  git --no-pager show --color "$COMMIT_HASH"
  echo -e "\nWould push & create PR with --apply:\n  git push -u origin $BRANCH\n  gh pr create --title \"$PR_TITLE\" --body <body>\n"
  exit 0
fi

# ---- Push & PR --------------------------------------------------------------

PR_BODY=$(cat <<EOF
This PR updates Go to the latest stable release.

* **go directive:** \`$GO_DIRECTIVE_VERSION\`
* **toolchain:** \`$TOOLCHAIN_VERSION\`
EOF
)

git push -u origin "$BRANCH"

gh pr create --title "$PR_TITLE" --body "$PR_BODY" --fill

echo "Done!"
