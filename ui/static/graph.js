(function () {
  'use strict';

  var app = document.getElementById('graph-app');
  if (!app) {
    return;
  }

  var stateURL = app.getAttribute('data-state-url') || '';
  var exportURL = app.getAttribute('data-export-url') || '';
  var settingsURL = app.getAttribute('data-settings-url') || '';
  var scanURL = app.getAttribute('data-scan-url') || '';
  var apiKey = app.getAttribute('data-api-key') || '';
  var panel = document.getElementById('graph-state-panel');
  var summaryEl = document.getElementById('graph-summary');
  var controls = document.getElementById('graph-controls');
  var legend = document.getElementById('graph-legend');
  var cyContainer = document.getElementById('cy');
  var detailEl = document.getElementById('node-detail');
  var exportBtn = document.getElementById('export-graph');
  var copyBtn = document.getElementById('copy-graph-summary');
  var refreshBtn = document.getElementById('graph-refresh');

  function headers() {
    var h = { Accept: 'application/json' };
    if (apiKey) {
      h['X-Repository-Detective-API-Key'] = apiKey;
    }
    return h;
  }

  function parseInitialState() {
    var el = document.getElementById('graph-initial-state');
    var raw = (el && el.textContent) ? el.textContent : '{}';
    try {
      return JSON.parse(raw);
    } catch (e) {
      return { state: 'missing' };
    }
  }

  function hideInteractive() {
    if (controls) controls.style.display = 'none';
    if (legend) legend.style.display = 'none';
    if (detailEl) detailEl.style.display = 'none';
    if (exportBtn) exportBtn.style.display = 'none';
    if (copyBtn) copyBtn.style.display = 'none';
    if (refreshBtn) refreshBtn.style.display = 'none';
    if (cyContainer) cyContainer.innerHTML = '';
    if (summaryEl) summaryEl.textContent = '';
  }

  function showInteractive() {
    if (controls) controls.style.display = '';
    if (legend) legend.style.display = '';
    if (detailEl) detailEl.style.display = '';
    if (exportBtn && exportURL) {
      exportBtn.href = exportURL;
      exportBtn.style.display = '';
    }
    if (copyBtn) copyBtn.style.display = '';
  }

  function actionLinks() {
    var parts = [];
    if (scanURL) {
      parts.push('<a class="rd-btn rd-btn-primary" href="' + scanURL + '">Run scan with graph enabled</a>');
    }
    if (settingsURL) {
      parts.push('<a class="rd-btn rd-btn-secondary" href="' + settingsURL + '">Open graph settings</a>');
    }
    return parts.join(' ');
  }

  function renderState(status) {
    if (!panel) {
      return;
    }
    hideInteractive();
    var state = (status && status.state) || 'missing';
    var html = '';
    switch (state) {
      case 'available':
      case 'truncated':
        panel.className = 'rd-alert rd-alert-success';
        panel.innerHTML = '<strong>Repository map available</strong>';
        if (state === 'truncated') {
          panel.className = 'rd-alert';
          panel.innerHTML = '<strong>Graph truncated</strong> — node/edge limits applied. ' +
            '<a class="rd-link" href="' + settingsURL + '">Graph settings</a>';
        } else {
          panel.innerHTML = '';
          panel.className = '';
        }
        showInteractive();
        if (state === 'pending' && refreshBtn) {
          refreshBtn.style.display = '';
        }
        renderGraph(status);
        return;
      case 'pending':
        panel.className = 'rd-alert';
        html = '<strong>Graph generation pending</strong><p>Analysis is still running. Refresh when the scan completes.</p>';
        if (refreshBtn) refreshBtn.style.display = '';
        break;
      case 'disabled':
        panel.className = 'rd-alert';
        html = '<strong>Code graph is disabled</strong><p>Enable code graph in repository settings or choose a deeper scan profile.</p>' +
          (status.failure_reason ? '<p class="rd-muted">' + status.failure_reason + '</p>' : '') +
          '<p>' + actionLinks() + '</p>';
        break;
      case 'failed':
        panel.className = 'rd-alert warn';
        html = '<strong>Graph generation failed</strong>' +
          (status.failure_reason ? '<p>' + status.failure_reason + '</p>' : '') +
          (status.next_action ? '<p>' + status.next_action + '</p>' : '') +
          '<p>' + actionLinks() + '</p>';
        break;
      case 'missing':
        panel.className = 'rd-alert';
        html = '<strong>No graph was stored for this scan.</strong><p>Run a new scan with code graph enabled and analysis depth ≥ 2.</p>' +
          '<p>' + actionLinks() + '</p>';
        break;
      case 'scan_not_found':
        panel.className = 'rd-alert warn';
        html = '<strong>Scan not found</strong><p>Open scan history and select a completed scan.</p>';
        break;
      case 'repo_not_found':
        panel.className = 'rd-alert warn';
        html = '<strong>Repository not found</strong>';
        break;
      default:
        panel.className = 'rd-alert warn';
        html = '<strong>Graph unavailable</strong>';
    }
    panel.innerHTML = html;
  }

  function buildSummary(g, status) {
    var m = (g && g.metrics) || {};
    var parts = [
      'Scan: ' + (status.scan_id || '—'),
      'Nodes: ' + (status.node_count || m.node_count || (g.nodes || []).length),
      'Edges: ' + (status.edge_count || m.edge_count || (g.edges || []).length)
    ];
    if (m.orphan_files != null) parts.push('Orphans: ' + m.orphan_files);
    if (m.findings_overlay != null) parts.push('Findings: ' + m.findings_overlay);
    if (status.truncated || m.truncated) parts.push('Truncated: yes');
    return parts.join(' · ');
  }

  function cssVar(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  function refreshGraphStyles(cy) {
    if (!cy) {
      return;
    }
    cy.style().fromJson(graphStyles()).update();
  }

  function graphStyles() {
    return [
      { selector: 'node', style: { label: 'data(label)', 'font-size': 10, color: cssVar('--rd-graph-label', '#e2e8f0'), 'text-valign': 'center', 'text-halign': 'center', 'background-color': cssVar('--rd-graph-edge', '#475569'), width: 24, height: 24 } },
      { selector: 'node[type="repo"]', style: { 'background-color': cssVar('--rd-graph-repo', '#6366f1'), width: 40, height: 40 } },
      { selector: 'node[type="directory"]', style: { 'background-color': cssVar('--rd-graph-dir', '#334155'), shape: 'round-rectangle' } },
      { selector: 'node[type="file"]', style: { 'background-color': cssVar('--rd-graph-node', '#64748b') } },
      { selector: 'node[type="package"]', style: { 'background-color': cssVar('--rd-graph-pkg', '#0ea5e9'), shape: 'hexagon' } },
      { selector: 'node[type="function"]', style: { 'background-color': cssVar('--rd-graph-fn', '#94a3b8'), width: 16, height: 16 } },
      { selector: 'node[type="finding"]', style: { 'background-color': cssVar('--rd-graph-finding', '#f59e0b'), shape: 'diamond' } },
      { selector: 'node[severity="critical"]', style: { 'background-color': cssVar('--rd-graph-sev-critical', '#dc2626') } },
      { selector: 'node[severity="high"]', style: { 'background-color': cssVar('--rd-graph-sev-high', '#ef4444') } },
      { selector: 'node[severity="medium"]', style: { 'background-color': cssVar('--rd-graph-sev-medium', '#f59e0b') } },
      { selector: 'node[severity="low"], node[severity="info"]', style: { 'background-color': cssVar('--rd-graph-sev-low', '#3b82f6') } },
      { selector: 'node[entrypoint]', style: { 'border-width': 3, 'border-color': cssVar('--rd-green', '#22c55e') } },
      { selector: 'node[disconnected]', style: { 'border-width': 2, 'border-color': cssVar('--rd-orange-hot', '#f97316'), 'border-style': 'dashed' } },
      { selector: 'node.highlight-entry', style: { 'border-width': 4, 'border-color': cssVar('--rd-green', '#22c55e') } },
      { selector: 'edge', style: { width: 1, 'line-color': cssVar('--rd-graph-edge', '#475569'), 'target-arrow-color': cssVar('--rd-graph-edge', '#475569'), 'target-arrow-shape': 'triangle', 'curve-style': 'bezier', opacity: 0.7 } }
    ];
  }

  function renderGraph(status) {
    var g = status.graph;
    if (!g || !g.nodes || g.nodes.length === 0) {
      if (summaryEl) summaryEl.textContent = 'Graph record exists but contains no nodes.';
      if (cyContainer) {
        cyContainer.innerHTML = '<p class="muted" style="padding:1rem">Empty graph payload.</p>';
      }
      return;
    }
    if (summaryEl) summaryEl.textContent = buildSummary(g, status);
    if (typeof cytoscape === 'undefined') {
      if (cyContainer) cyContainer.innerHTML = '<p class="warn">Graph library failed to load.</p>';
      return;
    }
    var elements = [];
    g.nodes.forEach(function (n) {
      elements.push({
        data: {
          id: n.id, label: n.label, type: n.type, path: n.path || '',
          severity: n.severity || '', category: n.category || '',
          disconnected: !!n.disconnected, entrypoint: !!n.entrypoint
        }
      });
    });
    (g.edges || []).forEach(function (e) {
      elements.push({ data: { id: e.id, source: e.from, target: e.to, type: e.type } });
    });
    var cy = cytoscape({
      container: cyContainer,
      elements: elements,
      style: graphStyles(),
      layout: { name: 'cose', animate: false, padding: 30 }
    });
    document.documentElement.addEventListener('rd-theme-change', function () {
      refreshGraphStyles(cy);
    });
    cy.on('tap', 'node', function (evt) {
      var d = evt.target.data();
      if (detailEl) {
        detailEl.innerHTML = '<strong>' + (d.label || d.id) + '</strong><br>' +
          ['Type: ' + d.type, d.path ? 'Path: ' + d.path : '', d.category ? 'Category: ' + d.category : '', d.severity ? 'Severity: ' + d.severity : ''].filter(Boolean).join('<br>');
      }
    });
    if (copyBtn) {
      copyBtn.onclick = function () {
        var text = buildSummary(g, status);
        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text);
        }
      };
    }
    window.__rdGraph = g;
  }

  function loadState() {
    if (!stateURL) {
      renderState(parseInitialState());
      return;
    }
    fetch(stateURL, { headers: headers(), credentials: 'same-origin' })
      .then(function (r) {
        if (r.status === 401 || r.status === 403) {
          throw new Error('Authentication required to load graph data');
        }
        return r.json().then(function (body) {
          if (!r.ok) {
            throw new Error(body.error || body.failure_reason || 'Graph API request failed');
          }
          return body;
        });
      })
      .then(function (status) {
        if (status.graph && typeof status.graph === 'string') {
          try { status.graph = JSON.parse(status.graph); } catch (e) { /* keep */ }
        }
        renderState(status);
      })
      .catch(function (e) {
        hideInteractive();
        if (panel) {
          panel.className = 'rd-alert warn';
          panel.innerHTML = '<strong>Could not load graph state</strong><p>' + (e.message || 'API error') + '</p>' +
            '<p>' + actionLinks() + '</p>';
        }
      });
  }

  if (refreshBtn) {
    refreshBtn.onclick = loadState;
  }

  loadState();
})();
