#!/usr/bin/env bash
# Migrate the Summit MongoDB data from a local/source server to MongoDB Atlas.
#
# Usage:
#   scripts/migrate-to-atlas.sh            # copy everything (drops target first)
#   scripts/migrate-to-atlas.sh -list      # list source collections only
#   scripts/migrate-to-atlas.sh -drop=false
#
# Environment:
#   SOURCE_MONGO_URI  source URI   (default: mongodb://admin:admin@localhost:27017)
#   PROD_MONGO_URI    Atlas URI    (default: read from .env)
#   MONGO_DB          database     (default: summit)
#
# The connection strings are never printed. The Atlas value in .env is
# normalised (some exports write "srv+mongodb://" and an extra "@>@").
set -euo pipefail

cd "$(dirname "$0")/.."

SOURCE_MONGO_URI="${SOURCE_MONGO_URI:-mongodb://admin:admin@localhost:27017}"
MONGO_DB="${MONGO_DB:-summit}"

# Load PROD_MONGO_URI from .env when not provided in the environment.
if [[ -z "${PROD_MONGO_URI:-}" && -f .env ]]; then
  PROD_MONGO_URI="$(grep -E '^PROD_MONGO_URI=' .env | head -1 | cut -d= -f2-)"
fi
if [[ -z "${PROD_MONGO_URI:-}" ]]; then
  echo "PROD_MONGO_URI is not set and not found in .env" >&2
  exit 1
fi

# Normalise common typos in the exported Atlas URI. Note: in ${var/pat/rep}
# the replacement is literal, so a "\/" would end up in the URI; replace the
# scheme prefix instead.
PROD_MONGO_URI="${PROD_MONGO_URI/#srv+mongodb:/mongodb+srv:}"
PROD_MONGO_URI="${PROD_MONGO_URI/@>@/@}"

echo "==> Building mongomigrate"
mkdir -p bin
go build -o bin/mongomigrate ./cmd/mongomigrate

echo "==> Running migration ($MONGO_DB, source -> Atlas)"
./bin/mongomigrate \
  -from "$SOURCE_MONGO_URI" \
  -to "$PROD_MONGO_URI" \
  -from-db "$MONGO_DB" \
  -to-db "$MONGO_DB" \
  "$@"
