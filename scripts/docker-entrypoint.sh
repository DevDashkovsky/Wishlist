#!/bin/sh
set -eu

if [ -z "${JWT_SECRET:-}" ]; then
    secret_file=/app/state/jwt-secret
    if [ ! -f "$secret_file" ]; then
        umask 077
        temporary=$(mktemp /app/state/.jwt-secret.XXXXXX)
        trap 'rm -f "$temporary"' EXIT
        secret=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
        [ "${#secret}" -eq 64 ] || exit 1
        printf '%s\n' "$secret" > "$temporary"
        if ! ln "$temporary" "$secret_file" 2>/dev/null; then
            [ -f "$secret_file" ] || exit 1
        fi
        rm -f "$temporary"
        trap - EXIT
    fi
    JWT_SECRET=$(cat "$secret_file")
    if [ "${#JWT_SECRET}" -ne 64 ]; then
        echo "Invalid persisted JWT secret in $secret_file" >&2
        exit 1
    fi
    export JWT_SECRET
fi

exec "$@"
