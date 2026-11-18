/* Repository Detective issue composer. No remote traffic or file uploads. */
(function (root) {
  'use strict';
  function redact(value) {
    return String(value || '')
      .replace(/-----BEGIN [\s\S]*?PRIVATE KEY-----[\s\S]*?-----END [\s\S]*?PRIVATE KEY-----/gi, '[REDACTED PRIVATE KEY]')
      .replace(/\b(?:ghp_|gho_|ghu_|ghs_|github_pat_|glpat-|gitea_)[A-Za-z0-9_\-]{12,}\b/g, '[REDACTED TOKEN]')
      .replace(/\beyJ[A-Za-z0-9_-]{12,}\.eyJ[A-Za-z0-9_-]{12,}\.[A-Za-z0-9_-]+\b/g, '[REDACTED JWT]')
      .replace(/((?:authorization\s*[:=]\s*)(?:bearer|basic)\s+)\S+/gi, '$1[REDACTED]')
      .replace(/((?:api[_-]?key|access[_-]?token|password|client_secret|secret)\s*[:=]\s*["']?)([^\s"'&,;]{5,})/gi, '$1[REDACTED]')
      .replace(/([?&](?:token|key|password|secret|api_key)=)[^\s&#]+/gi, '$1[REDACTED]');
  }
  function fence(text) { return '~~~text\n' + text.replace(/~~~/g, '~ ~ ~') + '\n~~~'; }
  function draft(values) {
    var title = String(values.title || '').trim().slice(0, 140);
    var steps = String(values.steps || '').trim().slice(0, 3000);
    if (!title || !steps) throw new Error('Add a title and reproduction steps.');
    if (values.logs && !values.logConsent) throw new Error('Review the log excerpt and check the consent box, or clear the logs.');
    var category = ['bug','scan','ux','docs','feature'].includes(values.category) ? values.category : 'bug';
    var lines = ['## Summary', '', redact(title), '', '## Reproduction steps', '', redact(steps), '', '## Expected behavior', '', redact(values.expected || 'Not provided'), '', '## Actual behavior', '', redact(values.actual || 'Not provided'), '', '## Environment', '', '- Application version/commit: ' + redact(values.version || 'Not provided'), '- Scan ID: ' + redact(values.scan || 'Not provided'), '- Feedback type: ' + category];
    if (values.logs) lines.push('', '## Reviewed log excerpt', '', fence(redact(String(values.logs).slice(0, 6000))));
    lines.push('', '> Submitted voluntarily by a Repository Detective user. This report does not establish that the issue is confirmed.', '');
    return { title: '[' + category.toUpperCase() + '] ' + redact(title), body: lines.join('\n') };
  }
  function issueLink(base, title, body) {
    var u = new URL(base.replace(/\/$/, '') + '/issues/new');
    u.searchParams.set('title', title);
    if (body.length <= 1800) u.searchParams.set('body', body);
    // Percent-encoding can make a short Unicode draft exceed practical URL limits.
    if (u.toString().length > 2800) u.searchParams.delete('body');
    return u.toString();
  }
  if (typeof module !== 'undefined' && module.exports) module.exports = {redact:redact, draft:draft, issueLink:issueLink};
  if (!root || !root.document) return;
  var document = root.document;
  function init() {
    var form = document.getElementById('rd-feedback-form');
    if (!form) return;
    function val(id) { var el = document.getElementById('rd-fb-' + id); return el ? el.value : ''; }
    var output = document.getElementById('rd-fb-markdown');
    var status = document.getElementById('rd-fb-status');
    var open = document.getElementById('rd-fb-open');
    function update() {
      try {
        var d = draft({category:val('category'), title:val('title'), steps:val('steps'), expected:val('expected'), actual:val('actual'), version:val('version'), scan:val('scan'), logs:val('logs'), logConsent:document.getElementById('rd-fb-log-consent').checked});
        output.value = d.body;
        if (open) {open.href = issueLink(form.dataset.forgeUrl, d.title, d.body);open.setAttribute('aria-disabled','false');}
        status.textContent = d.body.length > 1800 ? 'Preview ready. The report is too long to prefill: copy it, open Gitea, and paste it.' : 'Preview ready. Review every line before submitting it in Gitea.';
        return d;
      } catch (e) {
        status.textContent = e.message;
        if (open) {open.href = form.dataset.forgeUrl + '/issues/new';open.setAttribute('aria-disabled','true');}
        return null;
      }
    }
    document.getElementById('rd-fb-preview').addEventListener('click', update);
    document.getElementById('rd-fb-copy').addEventListener('click', function () {
      if (!update()) return;
      // Copy only after deliberate user activation; no automatic clipboard access.
      if (!navigator.clipboard || !navigator.clipboard.writeText) { output.focus();output.select();status.textContent = 'Select and copy the preview manually.';return; }
      navigator.clipboard.writeText(output.value).then(function () { status.textContent = 'Redacted draft copied. Review it again in Gitea before publishing.'; }, function () { output.focus();output.select();status.textContent = 'Select and copy the preview manually.'; });
    });
    if (open) open.addEventListener('click', function (event) {
      if (!update()) {event.preventDefault(); return;}
      if (open.getAttribute('aria-disabled') === 'true') event.preventDefault();
    });
  }
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init); else init();
}(typeof window === 'undefined' ? null : window));
