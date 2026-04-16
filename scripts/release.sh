#!/usr/bin/env bash
set -euo pipefail

die() {
	printf '%s\n' "$1" >&2
	exit 1
}

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

is_semver() {
	[[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

next_minor_version() {
	local latest_tag
	latest_tag=""
	for tag in $(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-version:refname); do
		latest_tag="$tag"
		break
	done

	if [[ -z "$latest_tag" ]]; then
		printf '0.1.0\n'
		return
	fi

	local base major minor patch
	base="${latest_tag#v}"
	if ! is_semver "$base"; then
		die "latest tag $latest_tag is not semver"
	fi

	IFS='.' read -r major minor patch <<<"$base"
	printf '%s.%s.0\n' "$major" "$((minor + 1))"
}

wait_for_release_run_id() {
	local tag="$1"
	local timeout_seconds poll_interval_seconds started_at run_id
	timeout_seconds="${RELEASE_WATCH_TIMEOUT_SECONDS:-180}"
	poll_interval_seconds="${RELEASE_WATCH_POLL_SECONDS:-3}"
	started_at="$(date +%s)"

	while true; do
		run_id="$(gh run list --workflow release.yml --event push --json databaseId,headBranch,displayTitle --limit 50 --jq ".[] | select(.headBranch == \"$tag\" or .displayTitle == \"$tag\") | .databaseId")"
		run_id="${run_id%%$'\n'*}"
		if [[ -n "$run_id" ]]; then
			printf '%s\n' "$run_id"
			return
		fi

		if (( $(date +%s) - started_at >= timeout_seconds )); then
			die "pushed $tag but could not find GitHub Actions release run within ${timeout_seconds}s"
		fi

		sleep "$poll_interval_seconds"
	done
}

version_input="${1:-}"
if [[ -n "$version_input" ]]; then
	version="${version_input#v}"
else
	version="$(next_minor_version)"
fi

if ! is_semver "$version"; then
	die "invalid version: $version (expected MAJOR.MINOR.PATCH)"
fi

require_cmd gh
gh auth status >/dev/null 2>&1 || die "gh is not authenticated (run: gh auth login)"
gh workflow view release.yml >/dev/null 2>&1 || die "cannot access GitHub Actions workflow release.yml"

tag="v$version"
if git rev-parse "$tag" >/dev/null 2>&1; then
	die "tag $tag already exists"
fi

if [[ -n "$(git status --porcelain)" ]]; then
	die "working tree is dirty; commit or stash changes before releasing"
fi

printf 'Releasing %s\n' "$tag"
git tag -a "$tag" -m "Release $tag"
git push origin "$tag"

run_id="$(wait_for_release_run_id "$tag")"
run_url="$(gh run view "$run_id" --json url --jq '.url')"
printf 'Watching release workflow run %s\n' "$run_url"

if gh run watch "$run_id" --exit-status; then
	printf 'Release workflow succeeded for %s\n' "$tag"
else
	die "release workflow failed for $tag ($run_url)"
fi
