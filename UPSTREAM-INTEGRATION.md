# Upstream integration: Seatsurfing 1.118.2 → 1.132.1

What changed when the CODYN fork was brought onto current upstream, why, and what
each change looks like in use.

Branch: `codyn-v2`, started from `upstream/main` (1.132.1).
Previous state preserved on `ByORG`, tagged `codyn-pre-upstream-merge`.

**Status: complete and green.** Backend, frontend and tests are ported; the app
builds and runs. Go suite (8 packages), UI unit tests (172) and the end-to-end
suite (7 specs, against a running instance) all pass.

---

## 1. Why this was a port, not a merge

Upstream replaced its whole permission system in PR #2549. These were **deleted**:

| Gone | What it was |
| --- | --- |
| `User.Role` field, `users.role` column | the ordered role ladder |
| super admin (role 90) | the cross-organization administrator |
| `IsSuperAdmin`, `IsSpaceAdmin` | role tests on the user repository |
| `CanManagePlatform`, `CanSpaceAdminOrg`, `CanAdminOrg` | router authorization helpers |
| `GET /organization/`, `POST /organization/` | listing and creating organizations |

In their place: **roles** holding **permissions** at a **level**
(`none` / `read` / `write` / `admin`), resolved per user per organization, plus an
`account_type` column that only selects how an account authenticates.

```go
// before                                  // after
GetUserRepository().IsSuperAdmin(user)      HasPermission(user, orgID,
CanAdminOrg(user, orgID)                        PermissionOrgSettings, PermissionLevelAdmin)
```

The CODYN platform layer — platform organization, client directory, organization
switcher — rested entirely on the deleted parts: **74 references** across four Go
files. `IsPlatformOrganization` queried `role >= 90`, which no longer compiles.
So the layer was rebuilt on the new model rather than merged into it.

---

## 2. The platform operator is now a role

**Decision:** express the operator inside upstream's model, not beside it.

- A new permission, `PermissionPlatform` (`"platform"`), governs running the
  service: the client directory and the organizations opened for clients.
- A seeded **system** role, `RoleNamePlatformOperator` ("Platform Operator"),
  grants it. It exists **only in the operator's own organization**.
- `CanManagePlatform(user)` is now a permission check. `IsPlatformOrganization(orgID)`
  asks who holds that permission instead of reading a dropped column.

### Why the permission is hidden from the catalogue

A system role grants *everything in the permission catalogue*. If `platform` were
in that catalogue, **every organization administrator in every client workspace
would silently become a platform operator**. So the permission is registered
(otherwise stored grants are discarded when access is resolved) but deliberately
left out of `builtInPermissions`. Three consequences, all wanted:

1. The org-admin role never picks it up.
2. It never appears in the Roles screen, so nobody can hand it out.
3. `POST`/`PUT /role/` rejects it outright, even though it would otherwise pass
   level validation.

### Scenario — a client admin tries to grant themselves the platform

Alice administers *Acme Workspace*. She has `roles: admin`, so she can create roles.

```http
POST /role/
Authorization: Bearer <alice>
{ "name": "Not Suspicious At All", "permissions": { "platform": 30 } }

→ 400 Bad Request
```

The Roles screen never offered `platform` in the first place; the server refuses
it anyway, because the UI is not the security boundary.

### Scenario — the operator is still walled out of client data

This is the invariant the fork exists to keep, and it is covered by tests.

```http
GET /location/{acme-location-id}      Authorization: Bearer <operator>   → 403
GET /user/{acme-staff-id}             Authorization: Bearer <operator>   → 403
GET /user/                            Authorization: Bearer <operator>
→ 200 [ ]        # the operator's own organization contains no staff, only
                 # client directory records — and operators are hidden from it
```

Platform power is **account-scoped, never data-scoped**: it governs the lifecycle
of client organizations (create, rename, domains, delete) and nothing inside them.

---

## 3. Your existing database: the upgrade trap we fixed

Upstream's migration converts a legacy super admin into **an administrator of
their own organization** — which is not the same thing as a platform operator.

Left alone, your existing install would have come out of the upgrade like this:

```
admin@seatsurfing.local
  before:  super admin  → client directory, org listing, workspace creation
  after:   org admin of CODYN Workspace → none of the above
```

Silently. No error, just a missing Clients page.

**Fixed:** legacy super admins are now also assigned the Platform Operator role,
and the role is seeded *only* in organizations that actually contain such a user —
so it never materialises in a client workspace for an admin there to assign.

Covered by `TestMigrationKeepsPlatformOperator`, which fails without the fix.

### Scenario — upgrading the database you have now

```bash
docker compose up -d --build        # schema 49 → 60+, migration runs once
```

Expected afterwards:

| Account | Outcome |
| --- | --- |
| `admin@seatsurfing.local` | org admin of CODYN Workspace **+ Platform Operator** |
| a client's org admin | org admin of their own workspace, nothing more |
| CODYN Workspace | still recognised as the platform organization |

Verify:

```bash
docker compose exec -T db psql -U seatsurfing -d seatsurfing -c \
  "SELECT u.email, r.name FROM users u
     JOIN user_roles ur ON ur.user_id = u.id
     JOIN roles r ON r.id = ur.role_id ORDER BY u.email;"
```

`admin@seatsurfing.local` must appear twice: once for
`Organization Administrator`, once for `Platform Operator`.

---

## 4. Endpoints: what changed

| Endpoint | State | Authorized by |
| --- | --- | --- |
| `GET /organization/` | **restored** (upstream deleted it) | operator → all client orgs; client → their own; else 403 |
| `POST /organization/` | **restored** | `platform: admin` |
| `POST /organization/my/` | kept | client opening their own workspace |
| `GET /user/organizations` | kept | any authenticated user (own memberships) |
| `POST /user/organizations/{id}/switch` | kept | same identity in the target org |
| `GET /user/clients` | kept | `platform: admin` |
| `POST /user/{id}/organizations` | kept | `platform: admin` |
| `DELETE /user/{id}/organizations/{orgId}` | kept | `platform: admin` |
| `DELETE /organization/{id}` | behaviour split | operator deletes outright; a client still needs the e-mailed code |

`GET /user/me` gained two booleans: `client` (this is the platform's own customer)
and `platform` (this is the operator). It also now returns upstream's
`permissions` map, which replaces the old `superAdmin` / `admin` / `spaceAdmin`
flags.

```json
{
  "id": "...",
  "email": "admin@seatsurfing.local",
  "platform": true,
  "client": false,
  "permissions": { "users": 30, "bookings": 30, "org_settings": 30, "platform": 30 },
  "roleIds": ["...", "..."]
}
```

### Scenario — a client opens a second workspace

Bob is a CODYN customer: a directory record in CODYN Workspace, and an admin
account in *Bob's Studio* sharing that e-mail and password hash.

```http
POST /organization/my/      Authorization: Bearer <bob>
{ "name": "Bob's Second Studio" }

→ 201 Created   X-Object-Id: <new-org-id>
```

What the server does: creates the organization, copies Bob's identity into it as
an administrator (**same password — no second credential**), seeds that
organization's built-in roles, and assigns Bob the org-admin role. If any step
fails the organization is deleted again, because a workspace nobody can
administer is worse than none.

```http
GET /user/organizations     Authorization: Bearer <bob>
→ [ { "organizationName": "Bob's Second Studio", "current": false },
    { "organizationName": "Bob's Studio",        "current": true  } ]
```

Note what is **absent**: CODYN Workspace. Bob has a record there, but it is
bookkeeping, not a workspace he owns, so it never appears in his switcher and he
cannot switch into it (`403`).

### Scenario — a promoted colleague cannot open a workspace

Bob promotes Carol to administrator of *Bob's Studio*. Carol runs that workspace
with him, but she is not a CODYN customer — there is no directory record for her.

```http
POST /organization/my/      Authorization: Bearer <carol>     → 403
```

A workspace Carol opened would answer to nobody the operator has on file. The
button is hidden from her too, but the server is what enforces it.

### Scenario — deleting a client takes their business with them

```http
DELETE /user/{bob-directory-record}      Authorization: Bearer <operator>
→ 204
```

Every workspace Bob runs goes with him — areas, bookings, staff. **Except** a
workspace he shares with another client: there, only Bob's own access is
withdrawn, so removing one customer never destroys another's data.

---

## 5. New upstream features, and how they land here

Seventeen upstream features arrived. These are the ones that touch CODYN.

### Custom roles (`/admin/roles/`)

Replaces the fixed admin tiers. A client can now define, say, a "Receptionist"
role with `bookings: admin` and nothing else.

Per-organization, so each client's roles are their own — and the operator cannot
see or edit them, because `roles` is checked against the caller's own
organization. The `platform` permission is never on the list.

### Authentication audit log (`/admin/audit/`)

Records every sign-in attempt per organization: who, when, which method, success
or failure.

Two things we changed here. **Organization switching is recorded** as method
`"organization switch"`, so a client moving between their workspaces leaves a
trail. And a **deleted account no longer disappears from the log** — the entry
survives with its `user_id` cleared, keeping the record of what happened without
naming somebody who no longer exists. That mirrors how mail logs are already
anonymised rather than destroyed.

### Public booking (`/book/`, `pages/book/*`)

Lets an organization take bookings from people with no account, confirmed by
e-mail, optionally with a map. Per-organization and off by default
(`PublicBookingEnabled`), so each client decides independently. Nothing
platform-wide.

### Feature flags replace single env vars

`ALLOW_ORG_DELETE=1` is deprecated; upstream reads `FEATURE_FLAGS` now.
Your `docker-compose.override.yaml` is already updated:

```yaml
FEATURE_FLAGS: "ALLOW_ORG_DELETE"      # was: ALLOW_ORG_DELETE: "1"
```

Valid flags: `ALLOW_ORG_DELETE`, `DOMAIN_VERIFICATION`. Note the flag governs an
organization deleting **itself**; the operator may delete a client workspace
regardless, since that is part of running the service.

### Other upstream additions, carried as-is

Booking calendar view for a user (`/admin/users/{id}/calendar`); hiding areas
from users not allowed to book them; buddy autocomplete; seat-proximity search;
custom alert and confirm dialogs; booking and buddy operations exposed to
plugins; improved domain administration; group members showing their role.

### Removed upstream, so removed here

The Confluence integration is gone, along with `MergeRequest` and the Confluence
login pages.

---

## 6. Upstream bugs fixed along the way

Both were caught by the fork's own isolation tests, and both are genuine defects
by any reading — rows left pointing at a user or organization about to vanish.

| Where | Was | Now |
| --- | --- | --- |
| `UserStore.Delete` | sessions and refresh tokens survived the user | both deleted with the account |
| `UserStore.DeleteAll` | sessions survived the organization | deleted with it |

A deleted account now keeps no way back in.

Two more, outside the deletion paths:

| Where | Was | Now |
| --- | --- | --- |
| `docker-compose.yaml` | moved to Postgres 18 but kept the 17 volume path, so the database would not start | mounted where 18 expects it |
| `BrowserUtil.test.ts` | asserted `de` and `zh-TW` are supported languages | says `en-GB` and `fr`, which is what this fork ships |

`clean-db.sh` was also stale: it knew none of the five tables upstream added, so
a wipe left a deleted tenant's roles behind and stripped the operator of theirs.
It now refuses to run at all when it meets a table it does not handle, so the
next upgrade reports that it is stale instead of quietly leaving data behind.

---

## 7. Decisions taken

| Question | Choice | Consequence |
| --- | --- | --- |
| Combine how? | Rebuild on upstream | Clean base, easier future updates; fork history stays on `ByORG` |
| Operator model? | Dedicated role + permission | Fits the new architecture; future merges stay clean |
| Theme? | Keep RoyalGlass, drop upstream's | Less work now; this conflict returns on each update |
| Languages? | `en-GB` + `fr` only | Upstream's 628 keys + your 35, re-reduced to two files |

---

## 8. The frontend

`RuntimeConfig` gains `client` and `platform` from the server, with
`isPlatformOperator()` and `isPlatformClient()` beside upstream's permission
helpers. The platform flag is read from its own field rather than the permission
map, because the permission is deliberately outside the catalogue.

The operator gets the platform's own menu — Overview, Clients, Organizations —
rather than a workspace menu with the desks removed, since bookings and areas
belong to a client's console and are scoped to that client's organization. A
client keeps the workspace menu plus their own organization list and switcher.

**Theme.** RoyalGlass is the theme of record. Upstream's own theming was removed
rather than kept alongside: both wrote `data-bs-theme`, but upstream's strips it
on `/admin`, holding the admin console to light — RoyalGlass is dark-capable
across the whole application, so the two could not both be right. Its three-way
selector gives way to the RoyalGlass switch; with no stored preference the
pre-paint script still follows the system, so automatic behaviour is kept.

Because RoyalGlass styles Bootstrap globally, **upstream's new screens — roles,
audit log, user calendar, public booking — pick up the design without being
touched individually.**

One regression worth recording: upstream deleted the `organizations` translation
key along with its own organization listing. This fork still has a listing, so
the sidebar and three pages rendered the bare key until it was restored. Three
further keys that upstream references from its own screens but ships in no
language file were filled in at the same time.

---

## 9. New client workspaces get a sample area

A workspace created for a client had no areas at all, so the client signed in to
an empty booking page and could do nothing until they had drawn a floor plan. A
new installation has never started that way — upstream gives its bootstrap
organization the sample floor — so both creation paths now do the same. The
platform organization is still left without one: nobody books a desk there.

---

## 10. Verifying it

Go is not installed on this machine, so both scripts run in containers:

```bash
./gocheck.sh          # compile the server
./gotest.sh           # full suite against a throwaway Postgres
./gotest.sh -run 'TestPlatform|TestClient' ./router/test/     # CODYN behaviour only
```

The UI and the end-to-end suite:

```bash
cd ui && npx vitest run                      # 172 unit tests
docker compose up -d --build                 # the app on :8080
cd e2e && UI_URL=http://localhost:8080 npx playwright test --project=chromium
```

Current state: **8 Go packages, 172 UI tests and 7 e2e specs pass**, including 22
CODYN behaviour tests and the migration test above.

The e2e suite signs in as a freshly provisioned *client*, not as the bootstrap
account: in this fork that account runs the platform and has no workspace, so a
client is the equivalent of the organization administrator upstream used.

The tests worth reading first, because they are the specification of this fork's
behaviour:

- `server/router/test/platform-isolation_test.go` — the operator cannot reach
  inside a client's workspace
- `server/router/test/client-organizations_test.go` — a client's workspaces share
  nothing but their identity
- `server/repository/test/role-migration_test.go` — an existing operator survives
  the upgrade
