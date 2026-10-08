#!/bin/sh

set -eu

PROJECT_ROOT=$(printenv PROJECT_ROOT 2>/dev/null || true)
[ -n "$PROJECT_ROOT" ] || PROJECT_ROOT=/ditoo/ditoo
ISULA_CONTAINER_PREFIX=$(printenv ISULA_CONTAINER_PREFIX 2>/dev/null || true)
DRY_RUN=0
OUTPUT_ARG=
WORK_DIR=
LOCK_DIR=
LOCK_ACQUIRED=0
ROLLBACK_NEEDED=0
BACKUP_DIR=
UPDATED_SERVICES=
UPDATED_SERVICES_FILE=
APPLIED_SERVICES_FILE=

usage() {
	cat <<'EOF'
Usage:
  update-project.sh [--dry-run] <output-directory>

Environment:
  PROJECT_ROOT            iSulad project root, default: /ditoo/ditoo
  ISULA_CONTAINER_PREFIX  Optional prefix added to every container name

Examples:
  sh scripts/update-project.sh /release/output
  PROJECT_ROOT=/ditoo/ditoo sh scripts/update-project.sh --dry-run /release/output
  ISULA_CONTAINER_PREFIX=dicloud- sh scripts/update-project.sh /release/output
EOF
}

die() {
	echo "[ERROR] $*" >&2
	exit 1
}

cleanup() {
	if [ "$LOCK_ACQUIRED" -eq 1 ]; then
		rm -f "$LOCK_DIR/pid"
		rmdir "$LOCK_DIR" 2>/dev/null || true
		LOCK_ACQUIRED=0
	fi
	if [ -n "$WORK_DIR" ] && [ -d "$WORK_DIR" ]; then
		rm -rf "$WORK_DIR"
	fi
}

container_name() {
	printf '%s%s\n' "$ISULA_CONTAINER_PREFIX" "$1"
}

restart_services() {
	restart_file=$1
	restart_phase=$2

	while IFS= read -r service; do
		[ -n "$service" ] || continue
		container=$(container_name "$service")
		echo "[$restart_phase] restarting $container"
		if ! isula restart "$container" >/dev/null; then
			echo "[$restart_phase-ERROR] failed to restart $container" >&2
			return 1
		fi
	done < "$restart_file"
}

assert_services_running() {
	check_file=$1

	while IFS= read -r service; do
		[ -n "$service" ] || continue
		container=$(container_name "$service")
		check_attempt=0
		container_state=
		while [ "$check_attempt" -lt 10 ]; do
			container_state=$(isula inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)
			[ "$container_state" = true ] && break
			check_attempt=$((check_attempt + 1))
			sleep 1
		done
		if [ "$container_state" != true ]; then
			echo "[ERROR] container is not running after update: $container" >&2
			return 1
		fi
		echo "[CHECK] $container is running"
	done < "$check_file"
}

rollback() {
	rollback_failed=0
	[ "$ROLLBACK_NEEDED" -eq 1 ] || return 0

	echo "[ROLLBACK] restoring files from $BACKUP_DIR"
	while IFS= read -r service; do
		[ -n "$service" ] || continue
		target_dir=$PROJECT_ROOT/project/$service
		target_file=$target_dir/$service
		backup_file=$BACKUP_DIR/$service

		if [ -f "$backup_file" ]; then
			if ! rollback_tmp=$(mktemp "$target_dir/.$service.rollback.XXXXXX"); then
				echo "[ROLLBACK-ERROR] cannot create temporary file for $service" >&2
				rollback_failed=1
				continue
			fi
			if ! cp "$backup_file" "$rollback_tmp" || ! chmod 0755 "$rollback_tmp" || ! mv -f "$rollback_tmp" "$target_file"; then
				rm -f "$rollback_tmp"
				echo "[ROLLBACK-ERROR] cannot restore $service" >&2
				rollback_failed=1
				continue
			fi
		elif ! rm -f "$target_file"; then
			echo "[ROLLBACK-ERROR] cannot remove new file for $service" >&2
			rollback_failed=1
			continue
		fi
		echo "[ROLLBACK] restored $service"
	done < "$APPLIED_SERVICES_FILE"

	if [ "$rollback_failed" -ne 0 ]; then
		echo "[ROLLBACK-ERROR] file rollback was incomplete" >&2
		return 1
	fi
	if ! restart_services "$APPLIED_SERVICES_FILE" ROLLBACK; then
		return 1
	fi
	if ! assert_services_running "$APPLIED_SERVICES_FILE"; then
		echo "[ROLLBACK-ERROR] one or more restored containers are not running" >&2
		return 1
	fi
	echo "[ROLLBACK] restored containers restarted"
}

finish() {
	status=$1
	trap - EXIT HUP INT TERM

	if [ "$status" -ne 0 ] && [ "$ROLLBACK_NEEDED" -eq 1 ]; then
		if rollback; then
			echo "[ROLLBACK] completed"
		else
			echo "[ROLLBACK] manual recovery may be required; backup: $BACKUP_DIR" >&2
		fi
	fi
	cleanup
	exit "$status"
}

trap 'finish $?' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

is_supported_service() {
	case "$1" in
		alarm|apiserver|auth|business|deal|mqtt|statistic|tolink|trigger|udp) return 0 ;;
		*) return 1 ;;
	esac
}

append_updated_service() {
	if [ -z "$UPDATED_SERVICES" ]; then
		UPDATED_SERVICES=$1
	else
		UPDATED_SERVICES="$UPDATED_SERVICES $1"
	fi
}

acquire_lock() {
	if mkdir "$LOCK_DIR" 2>/dev/null; then
		LOCK_ACQUIRED=1
		printf '%s\n' "$$" > "$LOCK_DIR/pid"
		return 0
	fi

	lock_pid=
	if [ -f "$LOCK_DIR/pid" ]; then
		lock_pid=$(cat "$LOCK_DIR/pid" 2>/dev/null || true)
	fi
	case "$lock_pid" in
		''|*[!0-9]*) ;;
		*)
			if kill -0 "$lock_pid" 2>/dev/null; then
				die "another project update is already running, pid: $lock_pid"
			fi
			;;
	esac

	echo "[LOCK] removing stale update lock: $LOCK_DIR"
	rm -f "$LOCK_DIR/pid"
	rmdir "$LOCK_DIR" 2>/dev/null || die "cannot remove stale update lock: $LOCK_DIR"
	mkdir "$LOCK_DIR" 2>/dev/null || die "cannot acquire update lock: $LOCK_DIR"
	LOCK_ACQUIRED=1
	printf '%s\n' "$$" > "$LOCK_DIR/pid"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--dry-run) DRY_RUN=1 ;;
		-h|--help)
			usage
			exit 0
			;;
		--*) die "unknown option: $1" ;;
		*)
			[ -z "$OUTPUT_ARG" ] || die "only one output directory may be provided"
			OUTPUT_ARG=$1
			;;
	esac
	shift
done

[ -n "$OUTPUT_ARG" ] || {
	usage >&2
	exit 1
}
[ -d "$PROJECT_ROOT" ] || die "project root does not exist: $PROJECT_ROOT"
[ -d "$OUTPUT_ARG" ] || die "output directory does not exist: $OUTPUT_ARG"
command -v isula >/dev/null 2>&1 || die "isula command was not found"
case "$ISULA_CONTAINER_PREFIX" in
	*[!A-Za-z0-9_.-]*) die "ISULA_CONTAINER_PREFIX contains unsupported characters" ;;
esac

if ! PROJECT_ROOT=$(CDPATH= cd "$PROJECT_ROOT" 2>/dev/null && pwd -P); then
	die "cannot resolve project root"
fi
if ! OUTPUT_DIR=$(CDPATH= cd "$OUTPUT_ARG" 2>/dev/null && pwd -P); then
	die "cannot resolve output directory: $OUTPUT_ARG"
fi
LOCK_DIR=$PROJECT_ROOT/.update-project.lock

TMP_ROOT=$(printenv TMPDIR 2>/dev/null || true)
[ -n "$TMP_ROOT" ] || TMP_ROOT=/tmp
WORK_DIR=$(mktemp -d "$TMP_ROOT/dicloud-project-update.XXXXXX") || die "cannot create temporary directory"
UPDATED_SERVICES_FILE=$WORK_DIR/updated-services
APPLIED_SERVICES_FILE=$WORK_DIR/applied-services
: > "$UPDATED_SERVICES_FILE"
: > "$APPLIED_SERVICES_FILE"

SERVICE_COUNT=0
for entry in "$OUTPUT_DIR"/* "$OUTPUT_DIR"/.[!.]* "$OUTPUT_DIR"/..?*; do
	if [ ! -e "$entry" ] && [ ! -L "$entry" ]; then
		continue
	fi

	service=$(basename "$entry")
	[ ! -L "$entry" ] || die "symbolic links are not allowed in output: $service"
	[ -f "$entry" ] || die "output entry is not a regular file: $service"
	[ -s "$entry" ] || die "output file is empty: $service"
	is_supported_service "$service" || die "unsupported service file: $service"

	target_dir=$PROJECT_ROOT/project/$service
	target_file=$target_dir/$service
	[ -d "$target_dir" ] || die "target directory does not exist: $target_dir"
	[ ! -L "$target_dir" ] || die "target directory must not be a symbolic link: $target_dir"
	if [ -e "$target_file" ] || [ -L "$target_file" ]; then
		[ ! -L "$target_file" ] || die "target file must not be a symbolic link: $target_file"
		[ -f "$target_file" ] || die "target path is not a regular file: $target_file"
	fi

	container=$(container_name "$service")
	isula inspect "$container" >/dev/null 2>&1 || die "iSulad container does not exist: $container"

	printf '%s\n' "$service" >> "$UPDATED_SERVICES_FILE"
	append_updated_service "$service"
	SERVICE_COUNT=$((SERVICE_COUNT + 1))
done

[ "$SERVICE_COUNT" -gt 0 ] || die "no supported service files found in output: $OUTPUT_DIR"

timestamp=$(date -u +%Y%m%d-%H%M%S)
backup_base=$PROJECT_ROOT/backup/project-update-$timestamp
BACKUP_DIR=$backup_base
backup_index=0
while [ -e "$BACKUP_DIR" ]; do
	backup_index=$((backup_index + 1))
	BACKUP_DIR="$backup_base-$backup_index"
done

echo "[PLAN] output: $OUTPUT_DIR"
echo "[PLAN] project root: $PROJECT_ROOT"
echo "[PLAN] services: $UPDATED_SERVICES"
echo "[PLAN] backup: $BACKUP_DIR"
echo "[PLAN] runtime: isula"
if [ -n "$ISULA_CONTAINER_PREFIX" ]; then
	echo "[PLAN] container prefix: $ISULA_CONTAINER_PREFIX"
fi

if [ "$DRY_RUN" -eq 1 ]; then
	echo "[DRY-RUN] no files will be changed and no containers will be restarted"
	while IFS= read -r service; do
		[ -n "$service" ] || continue
		echo "[DRY-RUN] isula restart $(container_name "$service")"
	done < "$UPDATED_SERVICES_FILE"
	exit 0
fi

acquire_lock
mkdir -p "$BACKUP_DIR" || die "cannot create backup directory: $BACKUP_DIR"

while IFS= read -r service; do
	[ -n "$service" ] || continue
	source_file=$OUTPUT_DIR/$service
	target_dir=$PROJECT_ROOT/project/$service
	target_file=$target_dir/$service
	backup_file=$BACKUP_DIR/$service

	if [ -e "$target_file" ] || [ -L "$target_file" ]; then
		cp -p "$target_file" "$backup_file" || die "cannot back up $service"
	else
		echo "[BACKUP] $service has no existing file; rollback will remove the new file"
	fi

	if ! update_tmp=$(mktemp "$target_dir/.$service.update.XXXXXX"); then
		die "cannot create temporary file for $service"
	fi
	if ! cp "$source_file" "$update_tmp" || ! chmod 0755 "$update_tmp" || ! mv -f "$update_tmp" "$target_file"; then
		rm -f "$update_tmp"
		die "cannot install $service"
	fi

	printf '%s\n' "$service" >> "$APPLIED_SERVICES_FILE"
	ROLLBACK_NEEDED=1
	echo "[UPDATE] $target_file"
done < "$UPDATED_SERVICES_FILE"

restart_services "$UPDATED_SERVICES_FILE" UPDATE || die "iSulad container restart failed"
assert_services_running "$UPDATED_SERVICES_FILE" || die "one or more updated containers are not running"

ROLLBACK_NEEDED=0
echo "[SUCCESS] updated $SERVICE_COUNT service(s)"
echo "[SUCCESS] backup: $BACKUP_DIR"
exit 0
