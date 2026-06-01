const selectedRepos = new Set();

function apiHeaders() {
  const key = document.getElementById('apiKey').value.trim();
  const headerName = 'X-Repository-Detective-API-Key';
  const legacyHeader = 'X-Bugbot-API-Key';
  if (!key) {
    return { 'Content-Type': 'application/json' };
  }
  return {
    [headerName]: key,
    [legacyHeader]: key,
    'Content-Type': 'application/json',
  };
}

function setStatus(elId, message, ok) {
  const el = document.getElementById(elId);
  el.textContent = message;
  el.className = 'status ' + (ok ? 'ok' : 'err');
}

function connectionPayload() {
  return {
    gitea_url: document.getElementById('giteaUrl').value.trim(),
    gitea_token: document.getElementById('giteaToken').value.trim(),
    public_url: document.getElementById('publicUrl').value.trim(),
    webhook_secret: document.getElementById('webhookSecret').value.trim(),
    ai_provider: document.getElementById('aiProvider').value,
    ai_base_url: document.getElementById('aiBaseUrl').value.trim(),
    ai_api_key: document.getElementById('aiApiKey').value.trim(),
    ai_model: document.getElementById('aiModel').value.trim(),
  };
}

function updateEnvExport() {
  const p = connectionPayload();
  const lines = [
    `REPOSITORY_DETECTIVE_GITEA_URL=${p.gitea_url}`,
    `REPOSITORY_DETECTIVE_GITEA_TOKEN=${p.gitea_token || 'your-token'}`,
    `REPOSITORY_DETECTIVE_WEBHOOK_SECRET=${p.webhook_secret || 'your-webhook-secret'}`,
    `REPOSITORY_DETECTIVE_API_KEY=${document.getElementById('apiKey').value.trim() || '<set-api-key>'}`,
    `REPOSITORY_DETECTIVE_PUBLIC_URL=${p.public_url}`,
    `REPOSITORY_DETECTIVE_AI_PROVIDER=${p.ai_provider}`,
    `REPOSITORY_DETECTIVE_AI_BASE_URL=${p.ai_base_url}`,
    `REPOSITORY_DETECTIVE_AI_API_KEY=${p.ai_api_key || 'your-ai-key'}`,
    `REPOSITORY_DETECTIVE_AI_MODEL=${p.ai_model}`,
    `REPOSITORY_DETECTIVE_ENABLE_SECURITY=true`,
    `REPOSITORY_DETECTIVE_ENABLE_QUALITY=true`,
    `REPOSITORY_DETECTIVE_AUTO_CREATE_ISSUES=true`,
    `REPOSITORY_DETECTIVE_LABEL_COMPAT_MODE=new_only`,
  ];
  document.getElementById('envExport').textContent = lines.join('\n');
}

async function loadDefaults() {
  try {
    const res = await fetch('/api/v1/onboard/defaults', { headers: apiHeaders() });
    if (!res.ok) return;
    const data = await res.json();
    if (data.gitea_url) document.getElementById('giteaUrl').value = data.gitea_url;
    if (data.public_url) document.getElementById('publicUrl').value = data.public_url;
    if (data.ai_provider) document.getElementById('aiProvider').value = data.ai_provider;
    if (data.ai_model) document.getElementById('aiModel').value = data.ai_model;
    if (data.webhook_url) document.getElementById('publicUrl').placeholder = data.webhook_url.replace('/webhook', '');
    updateEnvExport();
  } catch (_) {}
}

document.getElementById('testGiteaBtn').addEventListener('click', async () => {
  setStatus('giteaStatus', 'Testing…', true);
  try {
    const res = await fetch('/api/v1/onboard/test-gitea', {
      method: 'POST',
      headers: apiHeaders(),
      body: JSON.stringify(connectionPayload()),
    });
    const data = await res.json();
    setStatus('giteaStatus', data.message || (res.ok ? 'Connected' : 'Failed'), res.ok);
  } catch (e) {
    setStatus('giteaStatus', e.message, false);
  }
});

document.getElementById('testAiBtn').addEventListener('click', async () => {
  setStatus('aiStatus', 'Testing…', true);
  try {
    const res = await fetch('/api/v1/onboard/test-ai', {
      method: 'POST',
      headers: apiHeaders(),
      body: JSON.stringify(connectionPayload()),
    });
    const data = await res.json();
    setStatus('aiStatus', data.message || (res.ok ? 'Connected' : 'Failed'), res.ok);
  } catch (e) {
    setStatus('aiStatus', e.message, false);
  }
});

document.getElementById('loadReposBtn').addEventListener('click', async () => {
  const list = document.getElementById('repoList');
  list.innerHTML = '<p class="repo-meta">Loading…</p>';
  try {
    const res = await fetch('/api/v1/onboard/repos', {
      method: 'POST',
      headers: apiHeaders(),
      body: JSON.stringify(connectionPayload()),
    });
    const data = await res.json();
    if (!res.ok) {
      list.innerHTML = `<p class="status err">${data.error || 'Failed to load repos'}</p>`;
      return;
    }
    list.innerHTML = '';
    (data.repositories || []).forEach((repo) => {
      const id = repo.full_name;
      const div = document.createElement('div');
      div.className = 'repo-item';
      div.innerHTML = `
        <label>
          <input type="checkbox" data-repo="${id}" ${selectedRepos.has(id) ? 'checked' : ''}>
          <span><strong>${repo.full_name}</strong></span>
        </label>
        <span class="repo-meta">${repo.private ? 'private' : 'public'}${repo.description ? ' — ' + repo.description : ''}</span>
      `;
      div.querySelector('input').addEventListener('change', (e) => {
        if (e.target.checked) selectedRepos.add(id);
        else selectedRepos.delete(id);
      });
      list.appendChild(div);
    });
    if (!data.repositories || data.repositories.length === 0) {
      list.innerHTML = '<p class="repo-meta">No repositories found.</p>';
    }
  } catch (e) {
    list.innerHTML = `<p class="status err">${e.message}</p>`;
  }
});

document.getElementById('registerWebhooksBtn').addEventListener('click', async () => {
  if (selectedRepos.size === 0) {
    setStatus('webhookStatus', 'Select at least one repository', false);
    return;
  }
  setStatus('webhookStatus', 'Registering…', true);
  try {
    const payload = connectionPayload();
    payload.repositories = Array.from(selectedRepos);
    const res = await fetch('/api/v1/onboard/webhooks', {
      method: 'POST',
      headers: apiHeaders(),
      body: JSON.stringify(payload),
    });
    const data = await res.json();
    setStatus('webhookStatus', data.message || (res.ok ? 'Done' : 'Failed'), res.ok);
  } catch (e) {
    setStatus('webhookStatus', e.message, false);
  }
});

document.getElementById('copyEnvBtn').addEventListener('click', async () => {
  updateEnvExport();
  await navigator.clipboard.writeText(document.getElementById('envExport').textContent);
  setStatus('webhookStatus', 'Environment copied to clipboard', true);
});

['giteaUrl', 'giteaToken', 'publicUrl', 'webhookSecret', 'apiKey', 'aiProvider', 'aiBaseUrl', 'aiApiKey', 'aiModel'].forEach((id) => {
  document.getElementById(id).addEventListener('input', updateEnvExport);
});

loadDefaults();
updateEnvExport();
