# Authentication, roles, and repository access — beta security plan

## Available today

- **Local username/password** is implemented with `REPOSITORY_DETECTIVE_AUTH_MODE=local`, a required session secret, initial owner bootstrap, rate-limited login, CSRF protection, and server-side sessions. See [AUTH_LOCAL.md](AUTH_LOCAL.md). There is **no default username or password**.
- API keys remain a distinct automation identity. Do not put them in browser URLs.
- This release adds a defense-in-depth middleware check blocking read-only and unknown local roles from all protected UI mutation routes. This is **not complete RBAC**: fine-grained endpoint enforcement, API identities, forge permissions, and repository scoping still need exhaustive authorization tests.
- The current multi-user interface has edition-specific restrictions. Do not advertise free, multi-tenant RBAC or per-repository isolation until the product actually enforces it.

## Not implemented: external login

**Authentik OIDC and Gitea OAuth2 are design targets, not working login buttons.** An upstream reverse proxy protecting the site does not by itself make the app's API keys, webhook endpoints, and user permissions identity-aware.

Recommended design (no code should be enabled until tests below pass):

1. A single authorization-code + PKCE login flow, registered **per issuer**, with strict allowed callback URLs, one-time `state`, `nonce` for OIDC, and TLS. Authentik's application-scoped discovery document is at `/application/o/<slug>/.well-known/openid-configuration`; verify its actual `issuer` rather than guessing from the login host.
2. Gitea can act as an OAuth2 provider; use its explicit application registration and callback URL. OAuth claims must be correlated with a stable provider-scoped user subject (not a mutable display name or email alone).
3. Link identities to local principals explicitly. First-time external login should have **no elevated role by default**; provisioning and repository grants require authorized approval.
4. For each read/write API and UI route, enforce repository ID/owner permissions **server-side**, not by filtering the navigation menu. Worker, forge, and AI credentials require independent scoped identities.
5. Separate human sessions, automation keys, forge tokens, webhook secrets, and runner credentials. Rotate sessions and sensitive tokens on account disable, identity unlink, or compromise.
6. Add audit records for login, failed role checks, account linking, repository grant changes, exports, issue publication, remediation approval, and external PRs without logging secrets.

Required negative tests: forged/expired tokens, wrong issuer/audience, code replay, missing or mismatched `state`/`nonce`, disabled users, missing repository grants, privilege escalation, cross-tenant ID guessing, unauthorized exports, CSRF, session fixation, credential leaks, and browser logout. Run against disposable Authentik and Gitea providers before exposing to external testers.

Upstream documentation:
- https://docs.goauthentik.io/add-secure-apps/providers/oauth2/
- https://docs.gitea.com/development/oauth2-provider/
