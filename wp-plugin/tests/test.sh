#!/bin/sh
set -eu

root="$1"
for test in functions normalize meta_deny; do
	php -d error_reporting=-1 "$root/tests/$test.php"
done
