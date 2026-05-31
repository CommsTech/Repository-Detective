(function () {
  'use strict';

  var app = document.getElementById('graph-app');
  if (!app) {
    return;
  }

  var graphURL = app.getAttribute('data-graph-url') || '';
  var exportURL = app.getAttribute('data-export-url') || '';
  var apiKey = app.getAttribute('data-api-key') || '';
  var statusEl = document.getElementById('graph-status');
  var detailEl = document.getElementById('node-detail');
  var cyContainer = document.getElementById('cy');

  function headers() {
    return apiKey ? { 'X-Bugbot-API-Key': apiKey } : {};
  }

  function setStatus(text, isError) {
    if (!statusEl) {
      return;
    }
    statusEl.textContent = text;
    statusEl.className = isError ? 'warn' : 'muted';
  }

  function countByType(nodes, type) {
    var n = 0;
    (nodes || []).forEach(function (node) {
      if (node.type === type) {
        n++;
      }
    });
    return n;
  }

  function buildSummary(g) {
    var m = g.metrics || {};
    var parts = [
      'Nodes: ' + (m.node_count || (g.nodes || []).length),
      'Edges: ' + (m.edge_count || (g.edges || []).length),
      'Orphans: ' + (m.orphan_files || 0),
      'Findings: ' + (m.findings_overlay || countByType(g.nodes, 'finding')),
      'Entrypoints: ' + (m.entrypoint_count || 0)
    ];
    if (m.truncated) {
      parts.push('Truncated: yes');
    }
    return parts.join(' · ');
  }

  function copySummary(g) {
    var text = buildSummary(g);
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function () {
        setStatus(text + ' (copied to clipboard)');
      }).catch(function () {
        setStatus(text);
      });
      return;
    }
    setStatus(text);
  }

  if (!graphURL) {
    setStatus('Graph URL not configured', true);
    return;
  }

  fetch(graphURL, { headers: headers() })
    .then(function (r) {
      if (!r.ok) {
        throw new Error('Graph not available for this scan or repository');
      }
      return r.json();
    })
    .then(function (g) {
      var nodes = g.nodes || [];
      var edges = g.edges || [];
      if (nodes.length === 0) {
        setStatus('No graph data — run a scan at analysis depth ≥ 2 with code graph enabled.', true);
        if (cyContainer) {
          cyContainer.innerHTML = '<p class="muted" style="padding:1rem">Empty graph. Check repo settings and scan depth.</p>';
        }
        return;
      }

      setStatus(buildSummary(g));
      if (g.metrics && g.metrics.truncated) {
        var warn = document.getElementById('graph-truncated');
        if (warn) {
          warn.style.display = 'block';
        }
      }

      var elements = [];
      nodes.forEach(function (n) {
        elements.push({
          data: {
            id: n.id,
            label: n.label,
            type: n.type,
            path: n.path || '',
            severity: n.severity || '',
            category: n.category || '',
            disconnected: !!n.disconnected,
            entrypoint: !!n.entrypoint
          }
        });
      });
      (edges || []).forEach(function (e) {
        elements.push({ data: { id: e.id, source: e.from, target: e.to, type: e.type } });
      });

      if (typeof cytoscape === 'undefined') {
        setStatus('Graph library failed to load', true);
        return;
      }

      var cy = cytoscape({
        container: cyContainer,
        elements: elements,
        style: [
          { selector: 'node', style: { label: 'data(label)', 'font-size': 10, color: '#e2e8f0', 'text-valign': 'center', 'text-halign': 'center', 'background-color': '#475569', width: 24, height: 24 } },
          { selector: 'node[type="repo"]', style: { 'background-color': '#6366f1', width: 40, height: 40 } },
          { selector: 'node[type="directory"]', style: { 'background-color': '#334155', shape: 'round-rectangle' } },
          { selector: 'node[type="file"]', style: { 'background-color': '#64748b' } },
          { selector: 'node[type="package"]', style: { 'background-color': '#0ea5e9', shape: 'hexagon' } },
          { selector: 'node[type="function"]', style: { 'background-color': '#94a3b8', width: 16, height: 16 } },
          { selector: 'node[type="finding"]', style: { 'background-color': '#f59e0b', shape: 'diamond' } },
          { selector: 'node[severity="critical"]', style: { 'background-color': '#dc2626' } },
          { selector: 'node[severity="high"]', style: { 'background-color': '#ef4444' } },
          { selector: 'node[severity="medium"]', style: { 'background-color': '#f59e0b' } },
          { selector: 'node[severity="low"], node[severity="info"]', style: { 'background-color': '#3b82f6' } },
          { selector: 'node[entrypoint]', style: { 'border-width': 3, 'border-color': '#22c55e' } },
          { selector: 'node[disconnected]', style: { 'border-width': 2, 'border-color': '#f97316', 'border-style': 'dashed' } },
          { selector: 'node.highlight-entry', style: { 'border-width': 4, 'border-color': '#22c55e' } },
          { selector: 'edge', style: { width: 1, 'line-color': '#475569', 'target-arrow-color': '#475569', 'target-arrow-shape': 'triangle', 'curve-style': 'bezier', opacity: 0.7 } }
        ],
        layout: { name: 'cose', animate: false, padding: 30 }
      });

      cy.on('tap', 'node', function (evt) {
        var d = evt.target.data();
        if (detailEl) {
          detailEl.innerHTML = '<strong>' + (d.label || d.id) + '</strong><br>' +
            ['Type: ' + d.type, d.path ? 'Path: ' + d.path : '', d.category ? 'Category: ' + d.category : '', d.severity ? 'Severity: ' + d.severity : ''].filter(Boolean).join('<br>');
        }
      });

      function applyFilters() {
        var typeFilter = document.getElementById('filter-type');
        var disconnectedOnly = document.getElementById('filter-disconnected');
        var hideFindings = document.getElementById('filter-findings-off');
        var t = typeFilter ? typeFilter.value : '';
        cy.nodes().forEach(function (n) {
          var show = true;
          if (t && n.data('type') !== t) {
            show = false;
          }
          if (disconnectedOnly && disconnectedOnly.checked && !n.data('disconnected')) {
            show = false;
          }
          if (hideFindings && hideFindings.checked && n.data('type') === 'finding') {
            show = false;
          }
          n.style('display', show ? 'element' : 'none');
        });
      }

      function toggleEntrypoints() {
        var cb = document.getElementById('filter-entrypoints');
        if (!cb) {
          return;
        }
        cy.nodes().removeClass('highlight-entry');
        if (cb.checked) {
          cy.nodes('[?entrypoint]').addClass('highlight-entry');
        }
      }

      var layoutSelect = document.getElementById('layout-mode');
      if (layoutSelect) {
        layoutSelect.onchange = function () {
          cy.layout({ name: this.value, animate: true, padding: 30 }).run();
        };
      }
      ['filter-type', 'filter-disconnected', 'filter-findings-off'].forEach(function (id) {
        var el = document.getElementById(id);
        if (el) {
          el.onchange = applyFilters;
        }
      });
      var entryCb = document.getElementById('filter-entrypoints');
      if (entryCb) {
        entryCb.onchange = toggleEntrypoints;
      }

      var copyBtn = document.getElementById('copy-graph-summary');
      if (copyBtn) {
        copyBtn.onclick = function () { copySummary(g); };
      }

      window.__bugbotGraph = g;
    })
    .catch(function (e) {
      setStatus(e.message || 'Failed to load graph', true);
    });
})();
