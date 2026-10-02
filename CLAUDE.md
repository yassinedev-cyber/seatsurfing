# CLAUDE.md

Coding rules and conventions are in AGENTS.md (already loaded). Don't repeat them here. This file only covers this fork and this machine.

## What this fork is
Upstream Seatsurfing (desk booking, Go + Next.js) turned into a multi-tenant SaaS branded **CODYN**. Branch `ByORG`; upstream base is `22e6f356`. Run `git diff 22e6f356 --stat` to see everything the fork changed.

Domain model added on top of upstream:
- **Platform org** = the operator's own org (the first org created, whose admin is the super admin). It is not a workspace. It holds the operator account plus the **client directory** (one plain user row per client). See `IsPlatformOrganization` in [server/repository/user-repository.go](server/repository/user-repository.go). It is hidden from org lists and switchers.
- **Client** = a customer, i.e. someone with a directory entry in the platform org. `IsPlatformClient(user)` is in [server/router/routes.go](server/router/routes.go). `GET /user/me` returns `client: true` for them, read in the UI as `RuntimeConfig.INFOS.client`.
- **Identity spans orgs by email**: one person has one user row per org, all with the same email and password hash. `GetUsersWithEmail` resolves the person across orgs.
- Endpoints: `GET /user/organizations`, `POST /user/organizations/{id}/switch`, `GET /user/clients` and `POST|DELETE /user/{id}/organizations[/{orgId}]` (platform only, `CanManagePlatform`), `POST /organization/my/` (only a client can create a workspace; an org admin a client promoted cannot).
- Rule: orgs owned by the same client share **no** data. Tests for this: `server/router/test/platform-isolation_test.go` and `client-organizations_test.go`.
- UI: `pages/admin/clients/`, `pages/admin/overview.tsx`, `components/OrganizationSwitcher.tsx`, `BrandLogo.tsx`, `ThemeSwitch.tsx`. Theme is `styles/RoyalGlass.css`.
- i18n: only `en-GB` (master) and `fr` exist. The other languages were deleted on purpose. Add every new key to both files.

## Running on this machine (Windows)
- **Go is NOT installed**, so `server/run.sh` and `test.sh` don't work locally. Node 24 and Docker Desktop are installed.
- Full app: start Docker Desktop, then run `docker compose up -d --build` (base compose plus `docker-compose.override.yaml`, which sets CRYPT_KEY, the CODYN bootstrap and ALLOW_ORG_DELETE). Open http://localhost:8080/ui/ and log in as `admin@seatsurfing.local` / `Sea!surf1ng` (super admin).
- API smoke test: `GET /organization/domain/localhost` returns the org id, then `POST /auth/login {email,password,organizationId}` (organizationId is required).
- Rebuild after code changes: `docker compose up -d --build node`. Wipe the DB: `docker compose down -v`. Wipe everything except the super admin: `./clean-db.sh [-y]`.
- UI unit tests: `cd ui && npx vitest run`. UI dev server: `npm run dev` on :3000, which needs a backend.
- Go tests need Go plus Postgres (`seatsurfing_test` DB, postgres/root on localhost).

## Gotchas
- `server;C/` is an empty stray folder (an accident from a Windows path). Ignore it.
- Fork commits use messages like "dep4"; AGENTS.md asks for conventional commits. Use the conventional format for new commits.
