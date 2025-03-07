#!/bin/sh

if [ "$ENV" = "production" ]; then
  if [ -z "$TangoJWTSecret" ] || [ -z "$TangoTLSCert" ] || [ -z "$TangoTLSKey" ] || [ -z "$ENV" ]; then
    echo "Error: Required environment variable(s) (TangoJWTSecret, TangoTLSCert, TangoTLSKey) are not set in production." >&2
    exit 1
  fi
fi
exec "$@"
