#!/usr/bin/env bash
set -euo pipefail

die() {
	printf '%s\n' "$1" >&2
	exit 1
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

version_input="${1:-}"
if [[ -n "$version_input" ]]; then
	version="${version_input#v}"
else
	version="$(next_minor_version)"
fi

if ! is_semver "$version"; then
	die "invalid version: $version (expected MAJOR.MINOR.PATCH)"
fi

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
