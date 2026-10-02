#!/bin/bash
# Wipes all data from the docker compose database except the super admin login.
#
# Kept: the admin user, their organization (incl. domains and settings), their
# sessions, passkeys and preferences, and system settings (db_version, install_id).
# Everything else - other orgs, clients, users, locations, spaces, bookings,
# groups, auth providers, logs - is deleted in a single transaction.
#
# Usage: ./clean-db.sh [-y]        (-y skips the confirmation prompt)
# Env:   ADMIN_EMAIL (default admin@seatsurfing.local), DB_SERVICE, DB_USER, DB_NAME

set -euo pipefail

ADMIN_EMAIL="${ADMIN_EMAIL:-admin@seatsurfing.local}"
DB_SERVICE="${DB_SERVICE:-db}"
DB_USER="${DB_USER:-seatsurfing}"
DB_NAME="${DB_NAME:-seatsurfing}"

cd "$(dirname "$0")"

if [[ "${1:-}" != "-y" ]]; then
    read -r -p "Delete ALL data except the login '$ADMIN_EMAIL' from database '$DB_NAME'? [y/N] " answer
    [[ "$answer" =~ ^[yY]$ ]] || { echo "Aborted."; exit 1; }
fi

docker compose exec -T "$DB_SERVICE" psql -U "$DB_USER" -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 -v admin_email="$ADMIN_EMAIL" <<'SQL'
BEGIN;

CREATE TEMP TABLE keep_user ON COMMIT DROP AS
    SELECT id, organization_id FROM users
    WHERE LOWER(email) = LOWER(:'admin_email') AND role >= 90;

DO $$
BEGIN
    IF (SELECT COUNT(*) FROM keep_user) <> 1 THEN
        RAISE EXCEPTION 'Expected exactly one super admin with that email, found %. Nothing was deleted.',
            (SELECT COUNT(*) FROM keep_user);
    END IF;
END $$;

-- tables emptied completely
TRUNCATE auth_attempts, auth_states, auth_providers, mail_logs,
    bookings, recurring_bookings, buddies,
    users_groups, groups,
    spaces_allowed_bookers, spaces_approvers, locations_allowed_bookers,
    space_attribute_values, space_attributes,
    location_floor_plans, spaces, locations;

-- per-user tables: keep only the admin's rows
DELETE FROM passkeys          WHERE user_id::text NOT IN (SELECT id::text FROM keep_user);
DELETE FROM sessions          WHERE user_id::text NOT IN (SELECT id::text FROM keep_user);
DELETE FROM refresh_tokens    WHERE user_id::text NOT IN (SELECT id::text FROM keep_user);
DELETE FROM users_preferences WHERE user_id::text NOT IN (SELECT id::text FROM keep_user);
DELETE FROM users             WHERE id::text      NOT IN (SELECT id::text FROM keep_user);

-- per-org tables: keep only the admin's org (+ system settings under the nil UUID)
DELETE FROM organizations_domains WHERE organization_id::text NOT IN (SELECT organization_id::text FROM keep_user);
DELETE FROM settings WHERE organization_id::text NOT IN (SELECT organization_id::text FROM keep_user)
    AND organization_id::text <> '00000000-0000-0000-0000-000000000000';
DELETE FROM organizations WHERE id::text NOT IN (SELECT organization_id::text FROM keep_user);

COMMIT;

SELECT (SELECT COUNT(*) FROM organizations) AS organizations,
       (SELECT COUNT(*) FROM users)         AS users,
       (SELECT COUNT(*) FROM locations)     AS locations,
       (SELECT COUNT(*) FROM bookings)      AS bookings;
SQL

echo "Done. Remaining login: $ADMIN_EMAIL"
