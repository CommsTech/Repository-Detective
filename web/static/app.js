const selectedRepos = new Set();
let loadedRepos = [];
let defaultScanOrgs = [];
let lastVerifyReport = null;

function apiHeaders() {
  const key = document.getElementById('apiKey').value.trim();
  const h = { 'Content-Type': 'application/json' };
  if (key) h['X-Repository-Detective-API-Key'] = key;
  return h;
}

function setStatus(elId, message, ok) {
  const el = document.getElementById(elId);
  if (!el) return;
  el.textContent = message;
  el.className = 'status ' + (ok ? 'ok' : 'err');
}

function connectionPayload() {
  return {
    gitea_url: document.getElementById('giteaUrl').value.trim(),
    gitea_token: document.getElementById('giteaToken').value.trim(),
    public_url: document.getElementById('publicUrl').value.trim(),
    webhook_secret: document.getElementById('webhookSecret').value.trim(),
    privacy_mode: document.getElementById('privacyMode').value,
    scan_profile: document.getElementById('scanProfile').value,
    policy_mode: document.getElementById('policyMode').value,
    ai_provider: document.getElementById('aiProvider').value,
    ai_base_url: document.getElementById('aiBaseUrl').value.trim(),
    ai_api_key: document.getElementById('aiApiKey').value.trim(),
    ai_model: document.getElementById('aiModel').value.trim(),
    ai_enabled: !!document.getElementById('aiProvider').value,
    orgs: defaultScanOrgs.slice(),
    repositories: Array.from(selectedRepos),
    selected_repo: Array.from(selectedRepos)[0] || '',
  };
}

function showStage(n) {
  document.querySelectorAll('[data-stage-panel]').forEach((el) => {
    el.hidden = String(el.getAttribute('data-stage-panel')) !== String(n);
  });
  document.querySelectorAll('.rd-stage-nav [data-stage]').forEach((el) => {
    el.classList.toggle('active', String(el.getAttribute('data-stage')) === String(n));
  });
  updateEnvExport();
}

function updateEnvExport() {
  const p = connectionPayload();
  const lines = [
    `REPOSITORY_DETECTIVE_GITEA_URL=${p.gitea_url}`,
    `REPOSITORY_DETECTIVE_GITEA_TOKEN=${p.gitea_token || 'your-token'}`,
    `REPOSITORY_DETECTIVE_WEBHOOK_SECRET=${p.webhook_secret || 'your-webhook-secret'}`,
    `REPOSITORY_DETECTIVE_API_KEY=${document.getElementById('apiKey').value.trim() || '<set-api-key>'}`,
    `REPOSITORY_DETECTIVE_PUBLIC_URL=${p.public_url}`,
    `REPOSITORY_DETECTIVE_PRIVACY_MODE=${p.privacy_mode}`,
    `REPOSITORY_DETECTIVE_SCAN_PROFILE=${p.scan_profile}`,
    `REPOSITORY_DETECTIVE_ENABLE_LLM_AUDITORS=${p.ai_provider ? 'true' : 'false'}`,
    `# Policy mode ${p.policy_mode} — map in UI/repo settings (Observe/Warn/Enforce)`,
    `REPOSITORY_DETECTIVE_REJECT_QUERY_STRING_API_KEY=true`,
  ];
  if (p.ai_provider) {
    lines.push(
      `REPOSITORY_DETECTIVE_AI_PROVIDER=${p.ai_provider}`,
      `REPOSITORY_DETECTIVE_AI_BASE_URL=${p.ai_base_url}`,
      `REPOSITORY_DETECTIVE_AI_API_KEY=${p.ai_api_key || ''}`,
      `REPOSITORY_DETECTIVE_AI_MODEL=${p.ai_model}`,
    );
  }
  const el = document.getElementById('envExport');
  if (el) el.textContent = lines.join('\n');
}

function renderRepoList(repos) {
  const list = document.getElementById('repoList');
  const searchWrap = document.getElementById('repoSearchWrap');
  const searchInput = document.getElementById('repoSearch');
  loadedRepos = repos || [];
  list.innerHTML = '';
  if (!loadedRepos.length) {
    list.innerHTML = '<p class="repo-meta">No repositories found.</p>';
    if (searchWrap) searchWrap.hidden = true;
    return;
  }
  if (searchWrap) searchWrap.hidden = false;
  const query = ((searchInput && searchInput.value) || '').toLowerCase().trim();
  loadedRepos.filter((repo) => {
    if (!query) return true;
    return String(repo.full_name || '').toLowerCase().includes(query);
  }).forEach((repo) => {
    const id = repo.full_name;
    const div = document.createElement('div');
    div.className = 'repo-item';
    div.innerHTML = `
      <label>
        <input type="checkbox" data-repo="${id}" ${selectedRepos.has(id) ? 'checked' : ''}>
        <span><strong>${repo.full_name}</strong></span>
      </label>
      <span class="repo-meta">${repo.private ? 'private' : 'public'}${repo.language ? ' · ' + repo.language : ''}${repo.description ? ' — ' + repo.description : ''}</span>
    `;
    div.querySelector('input').addEventListener('change', (e) => {
      if (e.target.checked) selectedRepos.add(id);
      else selectedRepos.delete(id);
      updateEnvExport();
      maybeRecommend(repo);
    });
    list.appendChild(div);
  });
}

async function maybeRecommend(repo) {
  const hint = document.getElementById('profileHint');
  if (!hint || !repo) return;
  try {
    const res = await fetch('/api/v1/onboard/recommend-profile', {
      method: 'POST', headers: apiHeaders(),
      body: JSON.stringify({ language: repo.language || '', path_hints: [] }),
    });
    const data = await res.json();
    if (res.ok) {
      hint.textContent = `Recommended profile: ${data.profile} — ${data.reason}`;
      const sel = document.getElementById('scanProfile');
      if (sel && data.profile) sel.value = data.profile;
    }
  } catch (e) {
    console.warn(e);
  }
}

async function loadDefaults() {
  try {
    const res = await fetch('/api/v1/onboard/defaults', { headers: apiHeaders() });
    if (!res.ok) return;
    const d = await res.json();
    if (d.gitea_url) document.getElementById('giteaUrl').value = d.gitea_url;
    if (d.public_url) document.getElementById('publicUrl').value = d.public_url;
    if (d.privacy_mode) document.getElementById('privacyMode').value = d.privacy_mode;
    if (d.scan_profile) document.getElementById('scanProfile').value = d.scan_profile;
    if (d.policy_mode) document.getElementById('policyMode').value = d.policy_mode;
    if (d.ai_provider) document.getElementById('aiProvider').value = d.ai_provider;
    if (d.ai_base_url) document.getElementById('aiBaseUrl').value = d.ai_base_url;
    if (d.ai_model) document.getElementById('aiModel').value = d.ai_model;
    defaultScanOrgs = d.gitea_scan_orgs || [];
    updateEnvExport();
  } catch (err) {
    console.warn('Failed to load onboarding defaults', err);
  }
}

document.addEventListener('DOMContentLoaded', () => {
  loadDefaults();
  document.querySelectorAll('[data-next]').forEach((btn) => {
    btn.addEventListener('click', () => showStage(btn.getAttribute('data-next')));
  });
  document.querySelectorAll('.rd-stage-nav [data-stage]').forEach((el) => {
    el.addEventListener('click', () => showStage(el.getAttribute('data-stage')));
  });

  document.getElementById('testGiteaBtn').addEventListener('click', async () => {
    setStatus('giteaStatus', 'Testing…', true);
    try {
      const res = await fetch('/api/v1/onboard/test-gitea', {
        method: 'POST', headers: apiHeaders(), body: JSON.stringify(connectionPayload()),
      });
      const data = await res.json();
      setStatus('giteaStatus', data.message || data.error || res.statusText, res.ok);
    } catch (e) {
      setStatus('giteaStatus', String(e), false);
    }
  });

  document.getElementById('checkPermsBtn').addEventListener('click', async () => {
    const box = document.getElementById('permMatrix');
    box.hidden = false;
    box.textContent = 'Checking…';
    try {
      const res = await fetch('/api/v1/onboard/permissions', {
        method: 'POST', headers: apiHeaders(), body: JSON.stringify(connectionPayload()),
      });
      const data = await res.json();
      box.textContent = JSON.stringify(data, null, 2);
      setStatus('giteaStatus', res.ok ? 'Permission matrix loaded' : (data.error || 'failed'), res.ok);
    } catch (e) {
      box.textContent = String(e);
    }
  });

  document.getElementById('loadReposBtn').addEventListener('click', async () => {
    try {
      const res = await fetch('/api/v1/onboard/repos', {
        method: 'POST', headers: apiHeaders(), body: JSON.stringify(connectionPayload()),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || res.statusText);
      renderRepoList(data.repositories || []);
    } catch (e) {
      document.getElementById('repoList').innerHTML = `<p class="repo-meta err">${e}</p>`;
    }
  });

  const search = document.getElementById('repoSearch');
  if (search) search.addEventListener('input', () => renderRepoList(loadedRepos));

  document.getElementById('privacyPreviewBtn').addEventListener('click', async () => {
    const box = document.getElementById('privacyPreview');
    box.hidden = false;
    const p = connectionPayload();
    const res = await fetch('/api/v1/onboard/privacy-preview', {
      method: 'POST', headers: apiHeaders(),
      body: JSON.stringify({
        privacy_mode: p.privacy_mode, gitea_url: p.gitea_url,
        ai_provider: p.ai_provider, ai_base_url: p.ai_base_url, ai_enabled: p.ai_enabled,
      }),
    });
    const data = await res.json();
    box.textContent =
      `Privacy mode: ${data.privacy_mode}\n` +
      `AI endpoint: ${data.ai_endpoint || '(none)'} — ${data.ai_class} (allowed=${data.ai_allowed})\n` +
      `Notification endpoints: ${data.notifications}\n` +
      `Forge: ${data.forge_host || '(unset)'} — ${data.forge_class}\n\n` +
      `Result:\n${data.result}\n\n${data.guarantee_note}`;
  });

  document.getElementById('testAiBtn').addEventListener('click', async () => {
    const p = connectionPayload();
    if (!p.ai_provider) {
      setStatus('aiStatus', 'AI Analysis: Disabled — Status: VALID', true);
      return;
    }
    setStatus('aiStatus', 'Testing…', true);
    const res = await fetch('/api/v1/onboard/test-ai', {
      method: 'POST', headers: apiHeaders(), body: JSON.stringify(p),
    });
    const data = await res.json();
    setStatus('aiStatus', data.message || data.error || res.statusText, res.ok);
  });

  document.getElementById('registerWebhooksBtn').addEventListener('click', async () => {
    setStatus('webhookStatus', 'Registering…', true);
    const res = await fetch('/api/v1/onboard/webhooks', {
      method: 'POST', headers: apiHeaders(), body: JSON.stringify(connectionPayload()),
    });
    const data = await res.json();
    setStatus('webhookStatus', data.message || JSON.stringify(data), res.ok);
  });

  document.getElementById('runVerifyBtn').addEventListener('click', async () => {
    setStatus('verifyStatus', 'Running doctor checks…', true);
    const res = await fetch('/api/v1/onboard/verify', {
      method: 'POST', headers: apiHeaders(), body: JSON.stringify(connectionPayload()),
    });
    const data = await res.json();
    lastVerifyReport = data;
    const out = document.getElementById('verifyOut');
    if (!res.ok) {
      out.textContent = data.error || res.statusText;
      setStatus('verifyStatus', 'Verify failed', false);
      return;
    }
    const lines = [
      `Overall: ${data.overall}`,
      `Onboarding readiness: ${data.onboarding_ready}`,
      `Next: ${data.next_action}`,
      '',
    ];
    (data.report.checks || []).forEach((c) => {
      lines.push(`[${c.state}] ${c.id}: ${c.summary}`);
      if (c.detail) lines.push(`    ${c.detail}`);
    });
    lines.push('', data.first_scan_note || '');
    out.textContent = lines.join('\n');
    setStatus('verifyStatus', data.onboarding_ready, data.overall !== 'NOT_READY');
    const banner = document.getElementById('readyBanner');
    const summary = document.getElementById('readySummary');
    if (banner) {
      banner.textContent = `Repository Detective is ${data.onboarding_ready}`;
      banner.className = 'status ' + (data.overall === 'NOT_READY' ? 'err' : 'ok');
    }
    if (summary) {
      summary.textContent =
        `Overall: ${data.overall}\n` +
        `Profile: ${connectionPayload().scan_profile}\n` +
        `Policy mode: ${connectionPayload().policy_mode}\n` +
        `Privacy: ${connectionPayload().privacy_mode}\n` +
        `AI: ${connectionPayload().ai_provider || 'Disabled'}\n` +
        `Next action: ${data.next_action}`;
    }
  });

  document.getElementById('copyEnvBtn').addEventListener('click', async () => {
    updateEnvExport();
    await navigator.clipboard.writeText(document.getElementById('envExport').textContent);
  });

  ['giteaUrl', 'giteaToken', 'publicUrl', 'webhookSecret', 'apiKey', 'privacyMode', 'scanProfile', 'policyMode', 'aiProvider', 'aiBaseUrl', 'aiApiKey', 'aiModel']
    .forEach((id) => {
      const el = document.getElementById(id);
      if (el) el.addEventListener('input', updateEnvExport);
      if (el) el.addEventListener('change', updateEnvExport);
    });

  showStage(1);
  updateEnvExport();
});
