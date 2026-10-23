#!/bin/sh
# Runs the unit tests, prints coverage per package and fails when a package is below the minimum
# (first argument or MIN_COVERAGE, default 85).
# cmd/milbi is excluded: it only wires ldflags to cli.Execute.
set -u
min=${1:-${MIN_COVERAGE:-85}}

out=$(go test -cover ./... 2>&1)
status=$?
printf '%s\n' "$out"
if [ "$status" -ne 0 ]; then
	exit "$status"
fi

printf '%s\n' "$out" | awk -v min="$min" '
	/cmd\/milbi/ { next }
	{
		for (i = 1; i < NF; i++) {
			if ($i == "coverage:") {
				c = $(i + 1)
				sub(/%/, "", c)
				if (c ~ /^[0-9.]+$/ && c + 0 < min) {
					print "FAIL: " $2 " coverage " c "% is below " min "%"
					bad = 1
				}
			}
		}
	}
	END { exit bad }
'
