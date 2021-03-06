
#! /bin/bash

# Runs the linters against dendrite

# The linters can take a lot of resources and are slow, so they can be
# configured using two environment variables:
#
# - `DENDRITE_LINT_CONCURRENCY` - number of concurrent linters to run,
#   gometalinter defaults this to 8
# - `DENDRITE_LINT_DISABLE_GC` - if set then the the go gc will be disabled
#   when running the linters, speeding them up but using much more memory.


set -eu

args="--config=linter.json"

if [ -n "${MORPHEUS_LINT_CONCURRENCY:-}" ]
then args="$args --concurrency=$MORPHEUS_LINT_CONCURRENCY"
fi

if [ -z "${MORPHEUS_LINT_DISABLE_GC:-}" ]
then args="$args --enable-gc"
fi

echo "Installing lint search engine..."
go install github.com/alecthomas/gometalinter/
gometalinter --config=linter.json ./... --install

echo "Looking for lint..."
gometalinter ./... ${args}