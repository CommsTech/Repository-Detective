# Gitea wiki server repair plan

**Symptom:** HTTP 500 on `git clone` and `git push` to `https://git.commsnet.org/commstech/repository-detective.wiki.git`  
**Impact:** Marketing blocker; **does not block private beta** (docs ship in repo and beta package)  
**Product commit:** `6d011cf`

## Current status

| Check | Result |
|-------|--------|
| API `has_wiki` | **true** (repo settings) |
| Manual wiki page in UI | **unknown** — operator should create one |
| `git clone repository-detective.wiki.git` | **HTTP 500** |
| One-page push | **HTTP 500** (prior attempts) |
| Dry-run publish script | **PASS** — lists 23 local pages, no git ops |

## Expected wiki repo path (Gitea default)

On Gitea host, bare wiki repo typically at:

```text
{GITEA_DATA}/gitea-repositories/commstech/Repository-Detective.wiki.git
```

Exact path depends on `app.ini` `[repository] ROOT` and `[server] LFS_*` settings.

## Sanitized operator commands

```bash
# 1. Confirm wiki enabled (token redacted)
curl -s -H "Authorization: token $GITEA_TOKEN" \
  https://git.commsnet.org/api/v1/repos/commstech/Repository-Detective | jq '.has_wiki, .permissions'

# 2. Create one page in Gitea UI: Wiki → New Page → "Init-Test"

# 3. Clone (expect failure until fixed)
git clone "https://oauth2:${GITEA_TOKEN}@git.commsnet.org/commstech/repository-detective.wiki.git" /tmp/rd-wiki-test

# 4. One-page push test (after clone works)
cd /tmp/rd-wiki-test
echo "# Repair test $(date -Iseconds)" >> Repair-Test.md
git add Repair-Test.md && git commit -m "wiki repair test"
git push origin HEAD

# 5. Full publish (only after one-page push succeeds)
cd /path/to/repository-detective
./scripts/publish-gitea-wiki.sh
```

## Gitea logs to capture

During clone or push attempt:

```bash
journalctl -u gitea --since '10 min ago' -n 200
# or
tail -200 /var/log/gitea/gitea.log
```

Look for: missing bare repo, permission denied, hook failure, DB error on wiki metadata, storage mount read-only.

## Likely causes (ordered)

1. **Wiki bare repo never initialized** — fix: create first page in UI or `gitea admin regenerate hooks`
2. **Missing or corrupt `repository-detective.wiki.git` directory** — fix: recreate from Gitea admin docs
3. **Storage path permissions** — Gitea user cannot write `{ROOT}/commstech/`
4. **Reverse proxy / git HTTP backend misroute** — 500 on smart HTTP only for `.wiki.git`
5. **Server-side hook failure** — pre-receive/update hook error in log
6. **Gitea bug / version-specific** — check release notes; upgrade if known fix

## Safe repair steps

1. **Backup** existing wiki storage if directory exists
2. Create **one manual wiki page** in Gitea UI (initializes wiki repo)
3. Retry **clone** from operator workstation
4. If still 500, on Gitea host:
   - Verify directory exists and owner is `git` / `gitea`
   - Check disk space and inode availability
   - Run Gitea doctor: `gitea doctor check --run all` (on server)
5. If repo missing, use Gitea admin to recreate wiki repository for `commstech/Repository-Detective`
6. Retry one-page push
7. Run `./scripts/publish-gitea-wiki.sh` (not dry-run)

## Rollback

- Do not delete main `repository-detective.git` repo
- If wiki recreate fails, keep using in-repo `docs/` as fallback
- Restore wiki bare repo from backup if partial repair made things worse

## Private beta fallback

Testers use:

- `docs/` in repository
- `README_BETA.md` and `docs/beta/` in beta package
- No dependency on live Gitea wiki for onboarding

## Marketing gate

Wiki must populate or acceptable published fallback before public marketing. Current fallback (**in-repo docs**) is acceptable for **invited private beta**.
