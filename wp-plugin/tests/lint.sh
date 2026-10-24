#!/bin/sh
set -eu

root="$1"
version="$(php -r 'echo PHP_VERSION;')"
files="$(find "$root" -name '*.php' | sort)"
if [ -z "$files" ]; then
	echo "there is no PHP file under $root" >&2
	exit 1
fi

status=0
count=0
for file in $files; do
	count=$((count + 1))
	said="$(php -d error_reporting=-1 -d display_errors=1 -l "$file" 2>&1)" || status=1
	if [ "$said" != "No syntax errors detected in $file" ]; then
		echo "$said" >&2
		status=1
	fi
done

if [ "$status" -ne 0 ]; then
	echo "php -l on PHP $version found more than a clean parse" >&2
	exit 1
fi
echo "$count files parse clean on PHP $version"
