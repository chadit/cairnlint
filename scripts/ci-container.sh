#!/usr/bin/env bash
set -euo pipefail
trap 'echo "E: ci-container.sh failed on line ${LINENO}" >&2' ERR

# ci-container.sh - run one CI step inside a throwaway container.
#
# Starts a container from the pinned CI image, runs the given command against a
# bind-mounted checkout, and removes the container. Works the same on a
# self-hosted runner and on a developer machine, macOS included.
#
# Usage:
#   scripts/ci-container.sh run [options] -- COMMAND [ARG...]
#   scripts/ci-container.sh tag                 print the image tag for ci/Dockerfile
#   scripts/ci-container.sh build               rebuild the image even if it exists
#
# `run` builds the image when this host lacks it, so there is no separate build
# step to order. The tag is a hash of ci/Dockerfile, so editing it produces a
# new tag and the next run rebuilds, once per container store.
#
# Options:
#   --image REF     image to run (default: the tag derived from ci/Dockerfile)
#   --cpus N        cpu ceiling for the container (default: ${CI_CPUS:-4})
#   --memory SIZE   memory ceiling (default: ${CI_MEMORY:-6g})
#   --network MODE  container network: bridge, host, or none (default: bridge)
#   --env NAME      pass one environment variable through from the caller
#
# Environment:
#   CI_RUNTIME      container runtime to use; auto-detected when unset
#   CI_CACHE_DIR    host directory for Go, npm, and pip caches
#   CI_IMAGE        default image reference

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

DOCKERFILE="${REPO_ROOT}/ci/Dockerfile"
IMAGE=""
IMAGE_WAS_SET=0
CPUS="${CI_CPUS:-4}"
MEMORY="${CI_MEMORY:-6g}"
PIDS_LIMIT="${CI_PIDS:-2048}"
NETWORK="bridge"
CACHE_DIR="${CI_CACHE_DIR:-${HOME}/.cache/cairnlint-ci}"
PASS_ENV=()
CONTAINER_NAME=""

info() { echo "I: $*" >&2; }
warn() { echo "W: $*" >&2; }
die() {
	echo "E: $*" >&2
	exit 1
}

# usage prints this file's header block, so help and docs cannot drift apart.
usage() {
	awk 'NR < 5 { next } /^#/ { sub(/^# ?/, ""); print; next } { exit }' "${BASH_SOURCE[0]}"
	exit "${1:-0}"
}

# cleanup removes the container if it outlived the command. --rm normally has.
cleanup() {
	[[ -n "$CONTAINER_NAME" && -n "$RUNTIME" ]] || return 0
	"$RUNTIME" rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true # already gone is the common case
}
trap cleanup EXIT

# detect_runtime prints the container runtime to drive, preferring podman
# because that is what the runner hosts install.
detect_runtime() {
	if [[ -n "${CI_RUNTIME:-}" ]]; then
		printf '%s\n' "$CI_RUNTIME"
		return 0
	fi
	local candidate
	for candidate in podman docker; do
		if command -v "$candidate" >/dev/null 2>&1; then # suppress 'not found' noise
			printf '%s\n' "$candidate"
			return 0
		fi
	done
	die "no container runtime found. Install podman, or set CI_RUNTIME"
}

# Resolved on demand so `tag` works on a machine with no container runtime.
RUNTIME=""

# resolve_runtime fills RUNTIME the first time a container command needs it.
resolve_runtime() {
	[[ -n "$RUNTIME" ]] && return 0
	RUNTIME="$(detect_runtime)"
}

# rootless_podman is true when the runtime maps container uids into the calling
# user's namespace, which is the case the keep-id mapping is for.
rootless_podman() {
	[[ "$RUNTIME" == "podman" ]] || return 1
	local rootless
	rootless=$("$RUNTIME" info --format '{{.Host.Security.Rootless}}' 2>/dev/null) || return 1
	[[ "$rootless" == "true" ]]
}

# file_digest prints a hex digest, using whichever tool the platform has.
file_digest() {
	local path="$1"
	if command -v sha256sum >/dev/null 2>&1; then # suppress 'not found' noise
		sha256sum "$path" | cut -c1-12
		return 0
	fi
	if command -v shasum >/dev/null 2>&1; then # suppress 'not found' noise
		shasum -a 256 "$path" | cut -c1-12
		return 0
	fi
	die "no sha256sum or shasum found to derive the image tag"
}

# image_tag prints the image reference for the current Dockerfile. CI_IMAGE
# overrides it so a prebuilt or registry-hosted image can be used instead.
image_tag() {
	if [[ -n "${CI_IMAGE:-}" ]]; then
		printf '%s\n' "$CI_IMAGE"
		return 0
	fi
	[[ -f "$DOCKERFILE" ]] || die "missing ${DOCKERFILE}"
	printf 'cairnlint-ci:%s\n' "$(file_digest "$DOCKERFILE")"
}

# using_custom_image is true when the caller named an image. That image is
# theirs to supply, so `run` neither builds it nor forbids pulling it.
using_custom_image() {
	[[ -n "${CI_IMAGE:-}" || "$IMAGE_WAS_SET" == "1" ]]
}

# resolve_image fills IMAGE when the caller did not pass --image.
resolve_image() {
	[[ -n "$IMAGE" ]] && return 0
	IMAGE="$(image_tag)"
}

# image_present is true when the runtime already has the image locally.
image_present() {
	"$RUNTIME" image exists "$IMAGE" 2>/dev/null && return 0 # podman only, docker has no `image exists`
	"$RUNTIME" image inspect "$IMAGE" >/dev/null 2>&1        # docker fallback, absent image is expected
}

# build_image builds the CI image from ci/Dockerfile.
build_image() {
	resolve_runtime
	resolve_image
	info "building ${IMAGE} with ${RUNTIME}"
	"$RUNTIME" build -t "$IMAGE" -f "$DOCKERFILE" "${REPO_ROOT}/ci"
}

# ensure_image builds only when the image is absent.
ensure_image() {
	resolve_runtime
	resolve_image
	if image_present; then
		return 0
	fi
	if using_custom_image; then
		info "${IMAGE} is not local; leaving it to the runtime to fetch"
		return 0
	fi
	build_image
}

# id_args prints uid mapping flags. keep-id holds the caller's uid inside the
# container so workspace writes stay owned by the caller.
id_args() {
	if rootless_podman; then
		printf '%s\n' "--userns=keep-id"
	fi
	printf '%s\n' "--user"
	printf '%s:%s\n' "$(id -u)" "$(id -g)"
}

# run_step starts a container, runs one command in it, and removes it.
run_step() {
	local command=("$@")
	[[ ${#command[@]} -gt 0 ]] || die "run needs a command after --"

	ensure_image
	mkdir -p "$CACHE_DIR"
	CONTAINER_NAME="cairnlint-ci-$$-$(date +%s)"

	local args=(
		run --rm
		--name "$CONTAINER_NAME"
		--network "$NETWORK"
		--cpus "$CPUS"
		--memory "$MEMORY"
		--pids-limit "$PIDS_LIMIT"
		--security-opt no-new-privileges
		--cap-drop ALL
		--volume "${REPO_ROOT}:/work:z"
		--volume "${CACHE_DIR}:/cache:z"
		--workdir /work
		--env HOME=/tmp
		# git rejects the bind-mounted checkout as unsafe without this.
		--env GIT_CONFIG_COUNT=1
		--env GIT_CONFIG_KEY_0=safe.directory
		--env GIT_CONFIG_VALUE_0=/work
	)

	local mapping
	while IFS= read -r mapping; do
		args+=("$mapping")
	done < <(id_args)

	local name
	for name in ${PASS_ENV[@]+"${PASS_ENV[@]}"}; do
		args+=(--env "${name}=${!name-}")
	done

	# A derived tag only exists locally, so a miss means unbuilt, not unpulled.
	if ! using_custom_image; then
		args+=(--pull=never)
	fi

	args+=("$IMAGE" "${command[@]}")

	info "running in ${IMAGE} (cpus=${CPUS} memory=${MEMORY} network=${NETWORK})"
	"$RUNTIME" "${args[@]}"
}

parse_run_args() {
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--image)
			[[ $# -ge 2 ]] || die "--image requires a value"
			IMAGE="$2"
			IMAGE_WAS_SET=1
			shift 2
			;;
		--cpus)
			[[ $# -ge 2 ]] || die "--cpus requires a value"
			CPUS="$2"
			shift 2
			;;
		--memory)
			[[ $# -ge 2 ]] || die "--memory requires a value"
			MEMORY="$2"
			shift 2
			;;
		--network)
			[[ $# -ge 2 ]] || die "--network requires a value"
			NETWORK="$2"
			shift 2
			;;
		--env)
			[[ $# -ge 2 ]] || die "--env requires a variable name"
			PASS_ENV+=("$2")
			shift 2
			;;
		--)
			shift
			run_step "$@"
			return 0
			;;
		*) die "unknown option: $1" ;;
		esac
	done
	die "run needs a command after --"
}

main() {
	local action="${1:-}"
	shift || true
	case "$action" in
	tag) image_tag ;;
	build) build_image ;;
	run) parse_run_args "$@" ;;
	-h | --help | "") usage 0 ;;
	*) die "unknown action: ${action}" ;;
	esac
}

main "$@"
