import re

with open('incident_drilldown.html', 'r') as f:
    content = f.read()

# Replace CSS
css_pattern = re.compile(r'<style>.*?</style>', re.DOTALL)
new_css = """<style>
        .drilldown-container { max-width: 1400px; margin: 0 auto; padding: 1.5rem; }
        .header-actions { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
        .back-btn { background: rgba(255,255,255,0.05); border: 1px solid var(--border-color); color: var(--text-primary); padding: 8px 16px; border-radius: 6px; cursor: pointer; text-decoration: none; display: inline-flex; align-items: center; gap: 8px; font-weight: 500; font-size: 0.9rem; }
        .back-btn:hover { background: rgba(255,255,255,0.1); }
        
        .incident-header-card { background: var(--surface); border: 1px solid var(--border-color); border-radius: var(--radius-lg); padding: 1.5rem 2rem; margin-bottom: 2rem; display: flex; justify-content: space-between; align-items: center; box-shadow: 0 4px 12px rgba(0,0,0,0.1); }
        .header-main-info { display: flex; flex-direction: column; gap: 8px; }
        .header-kpis { display: flex; gap: 3rem; }
        .header-kpi-block { display: flex; flex-direction: column; gap: 4px; border-left: 1px solid var(--border-color); padding-left: 20px; }
        .header-kpi-block:first-child { border-left: none; padding-left: 0; }
        
        .drilldown-body { display: flex; gap: 1.5rem; }
        .main-panel { flex: 1; display: flex; flex-direction: column; gap: 1.5rem; }
        .side-panel { width: 400px; background: var(--surface); border: 1px solid var(--border-color); border-radius: var(--radius-lg); padding: 1.5rem; height: fit-content; }
        
        .property-table-card { background: var(--surface); border: 1px solid var(--border-color); border-radius: var(--radius-lg); overflow: hidden; }
        .property-table-header { background: rgba(255,255,255,0.02); padding: 12px 1.5rem; border-bottom: 1px solid var(--border-color); font-weight: 600; text-transform: uppercase; font-size: 0.85rem; color: var(--text-secondary); letter-spacing: 0.5px; }
        .property-table { width: 100%; border-collapse: collapse; }
        .property-table td { padding: 14px 1.5rem; border-bottom: 1px solid rgba(255,255,255,0.05); font-size: 0.9rem; vertical-align: top; }
        .property-table tr:hover { background: rgba(255,255,255,0.01); }
        .property-table td:first-child { width: 25%; background: rgba(255,255,255,0.01); color: var(--text-secondary); font-weight: 500; border-right: 1px solid rgba(255,255,255,0.02); }
        .property-table tr:last-child td { border-bottom: none; }

        .timeline { position: relative; padding-left: 20px; margin-top: 1rem; }
        .timeline::before { content: ''; position: absolute; top: 0; bottom: 0; left: 4px; width: 2px; background: var(--border-color); }
        .timeline-item { position: relative; margin-bottom: 2rem; }
        .timeline-item:last-child { margin-bottom: 0; }
        .timeline-item::before { content: ''; position: absolute; width: 12px; height: 12px; border-radius: 50%; background: var(--surface); border: 2px solid var(--text-secondary); left: -21px; top: 2px; z-index: 2; }
        .timeline-item.active::before { border-color: var(--c-critical); background: var(--c-critical); box-shadow: 0 0 0 4px rgba(239, 68, 68, 0.2); }
        .timeline-item.closed::before { border-color: #10B981; background: #10B981; box-shadow: 0 0 0 4px rgba(16, 185, 129, 0.2); }

        .tabs-nav { display: flex; gap: 4px; border-bottom: 1px solid var(--border-color); margin-bottom: -1px; }
        .tab-btn { background: transparent; padding: 12px 20px; color: var(--text-secondary); font-weight: 600; font-size: 0.85rem; border: none; border-bottom: 2px solid transparent; cursor: pointer; text-transform: uppercase; transition: all 0.2s; }
        .tab-btn:hover { color: var(--text-primary); }
        .tab-btn.active { color: var(--accent-purple); border-bottom-color: var(--accent-purple); }
        
        .badge { padding: 4px 10px; border-radius: 4px; font-size: 0.75rem; font-weight: 600; text-transform: uppercase; display: inline-flex; align-items: center; gap: 6px; }
        .badge-critical { background: rgba(239, 68, 68, 0.1); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.2); }
        .badge-warning { background: rgba(245, 158, 11, 0.1); color: #f59e0b; border: 1px solid rgba(245, 158, 11, 0.2); }
        .badge-info { background: rgba(59, 130, 246, 0.1); color: #3b82f6; border: 1px solid rgba(59, 130, 246, 0.2); }
    </style>"""
content = css_pattern.sub(new_css, content)

# Replace JS logic
js_pattern = re.compile(r'// Setup summary card\s*let html = `.*?document\.getElementById\(\'drilldownContent\'\)\.innerHTML = html;', re.DOTALL)
new_js = """// Setup UI based on NNMi-like layout
            const getSeverityColor = (sev) => {
                const s = sev.toLowerCase();
                if(s==='critical' || s==='down') return '#ef4444';
                if(s==='warning' || s==='degraded') return '#f59e0b';
                return '#3b82f6';
            };
            const sevBadgeClass = latest.severity === 'critical' ? 'badge-critical' : (latest.severity === 'warning' ? 'badge-warning' : 'badge-info');

            let html = `
                <div class="incident-header-card">
                    <div class="header-main-info">
                        <div style="color: var(--text-secondary); font-size: 0.85rem; font-family: monospace; text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 4px;">Incident Drilldown</div>
                        <div style="display:flex; align-items:center; gap: 16px; margin-bottom: 8px;">
                            <h2 style="margin:0; font-size: 1.8rem; color: var(--text-primary);">${latest.id}</h2>
                            <span class="badge ${sevBadgeClass}"><i class="fa-solid fa-triangle-exclamation"></i> ${latest.severity}</span>
                        </div>
                        <div style="color: var(--text-secondary); font-size: 0.9rem; font-family: monospace;">Group Key: ${groupKey}</div>
                    </div>
                    
                    <div class="header-kpis">
                        <div class="header-kpi-block">
                            <div style="color: var(--text-secondary); font-size: 0.8rem; text-transform: uppercase; font-weight: 600;">Status</div>
                            <div style="font-size: 1.1rem; font-weight: 600; color: ${activeStatus==='Open' ? 'var(--c-critical)' : '#10B981'}; display:flex; align-items:center; gap:6px;">
                                <div style="width:8px; height:8px; border-radius:50%; background:${activeStatus==='Open' ? 'var(--c-critical)' : '#10B981'}; box-shadow: 0 0 8px ${activeStatus==='Open' ? 'var(--c-critical)' : '#10B981'};"></div>
                                ${activeStatus}
                            </div>
                        </div>
                        <div class="header-kpi-block">
                            <div style="color: var(--text-secondary); font-size: 0.8rem; text-transform: uppercase; font-weight: 600;">Lifecycle</div>
                            <div style="font-size: 1.1rem; font-weight: 500; color: var(--text-primary);">${activeLifecycle}</div>
                        </div>
                        <div class="header-kpi-block">
                            <div style="color: var(--text-secondary); font-size: 0.8rem; text-transform: uppercase; font-weight: 600;">Source Node</div>
                            <div style="font-size: 1.1rem; font-weight: 500; color: var(--text-primary);">${latest.target}</div>
                        </div>
                    </div>
                </div>

                <div class="tabs-nav">
                    <button class="tab-btn active">General</button>
                    <button class="tab-btn">Correlated Incidents</button>
                    <button class="tab-btn">Source Node Details</button>
                    <button class="tab-btn">Diagnostics</button>
                </div>

                <div class="drilldown-body" style="margin-top: 1.5rem;">
                    <div class="main-panel">
                        <div class="property-table-card">
                            <div class="property-table-header">Incident Details</div>
                            <table class="property-table">
                                <tbody>
                                    <tr><td>ID</td><td style="font-family:monospace; color:var(--text-secondary);">${latest.uniqueKey || latest.id}</td></tr>
                                    <tr><td>Title</td><td style="font-weight:500;">${latest.id}</td></tr>
                                    <tr><td>Severity</td><td><span style="color:${getSeverityColor(latest.severity)}; font-weight:600; text-transform:capitalize;">${latest.severity}</span></td></tr>
                                    <tr><td>Status</td><td style="font-weight:500;">${activeStatus}</td></tr>
                                    <tr><td>Lifecycle State</td><td><span style="background: rgba(255,255,255,0.05); padding: 4px 8px; border-radius: 4px; font-size:0.8rem;">${activeLifecycle}</span></td></tr>
                                    <tr><td>Priority</td><td>Medium</td></tr>
                                    <tr><td>Assigned To</td><td><i class="fa-solid fa-user" style="color:var(--accent); font-size: 0.9rem; margin-right:6px;"></i>${assignedTo}</td></tr>
                                    <tr><td>Message</td><td style="line-height: 1.6;">${latest.description}</td></tr>
                                </tbody>
                            </table>
                        </div>
                        
                        <div class="property-table-card">
                            <div class="property-table-header">Origin & Analysis</div>
                            <table class="property-table">
                                <tbody>
                                    <tr><td>Family</td><td>Network Event <span style="color:var(--text-secondary); font-size:0.8rem;">(Auto-Detected)</span></td></tr>
                                    <tr><td>Origin</td><td>Hawk-Eye Telemetry</td></tr>
                                    <tr><td>Creation Time</td><td>${latest.openedTime}</td></tr>
                                    <tr><td>Total Occurrences</td><td><span style="font-weight:600; color:var(--accent);">${incidents.length}</span></td></tr>
                                </tbody>
                            </table>
                        </div>
                    </div>

                    <div class="side-panel">
                        <h3 style="margin-top:0; margin-bottom: 1.5rem; font-size: 0.9rem; color: var(--text-secondary); font-weight: 600; letter-spacing: 0.5px; text-transform:uppercase; border-bottom: 1px solid var(--border-color); padding-bottom: 12px;">Incident Timeline</h3>
                        <div class="timeline">
            `;

            incidents.forEach((inc, idx) => {
                const imeta = savedMeta[inc.uniqueKey] || {};
                const st = imeta.status || inc.status;
                
                let tlClass = 'timeline-item';
                if (st === 'Open') tlClass += ' active';
                else tlClass += ' closed';

                html += `
                            <div class="${tlClass}">
                                <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom: 6px;">
                                    <div style="font-weight: 600; font-size: 0.85rem; color: var(--text-primary);">${inc.lifecycle || (st==='Open'?'Triggered':'Resolved')}</div>
                                    <div style="font-size: 0.75rem; color: var(--text-secondary);">${inc.openedTime.split(',')[1] || inc.openedTime}</div>
                                </div>
                                <div style="font-size: 0.8rem; color: var(--text-secondary); margin-bottom: 8px;">
                                    <strong>Status:</strong> <span style="color:${st === 'Open' ? 'var(--c-critical)' : '#10B981'}; font-weight:500;">${st}</span>
                                </div>
                                <div style="font-size: 0.85rem; color: var(--text-primary); line-height: 1.5; background: rgba(255,255,255,0.02); padding: 8px; border-radius: 6px; border: 1px solid rgba(255,255,255,0.05);">${inc.description}</div>
                                ${(st === 'Closed' && inc.closureNotes) ? `<div style="margin-top: 8px; font-size: 0.8rem; color: #10B981; background: rgba(16, 185, 129, 0.1); padding: 8px; border-radius: 6px; border: 1px solid rgba(16, 185, 129, 0.2);"><i class="fa-solid fa-check-circle"></i> ${inc.closureNotes}</div>` : ''}
                            </div>
                `;
            });

            html += `
                        </div>
                    </div>
                </div>
            `;

            document.getElementById('drilldownContent').innerHTML = html;"""
content = js_pattern.sub(new_js, content)

with open('incident_drilldown.html', 'w') as f:
    f.write(content)
