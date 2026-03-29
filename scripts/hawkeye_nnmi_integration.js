'use strict';

/**
 * HawkEye NNMi Integration Widget
 * Displays NNMi incidents and topology in HawkEye dashboard
 */
bvdPluginManager.registerWidget({
    id: 'hawkeye_nnmi_integration',
    displayName: 'HawkEye NNMi Integration',
    
    init: function(ctx) {
        const widgetGroup = ctx.svgGroup;
        
        // Hide placeholder
        ctx.placeHolder.attr('style', 'visibility: hidden;');
        
        // Create widget container
        const container = widgetGroup
            .append('foreignObject')
            .attr('width', ctx.bbox.width)
            .attr('height', ctx.bbox.height)
            .attr('x', ctx.bbox.x)
            .attr('y', ctx.bbox.y)
            .append('xhtml:div')
            .attr('class', 'hawkeye-nnmi-widget')
            .style('width', '100%')
            .style('height', '100%')
            .style('overflow', 'auto')
            .style('background', '#1e1e2f')
            .style('color', '#fff')
            .style('border-radius', '4px')
            .style('padding', '10px');
        
        // Initial content
        container.html(`
            <div style="display: flex; flex-direction: column; height: 100%;">
                <h3 style="margin: 0 0 10px 0; color: #00bcd4;">NNMi Incidents</h3>
                <div id="incident-list" style="flex: 1; overflow-y: auto;">
                    Loading incidents...
                </div>
                <div style="margin-top: 10px; display: flex; gap: 5px;">
                    <button id="refresh-incidents" style="flex: 1; padding: 5px; background: #00bcd4; border: none; border-radius: 4px; color: white; cursor: pointer;">
                        Refresh
                    </button>
                    <button id="hawkeye-details" style="flex: 1; padding: 5px; background: #4caf50; border: none; border-radius: 4px; color: white; cursor: pointer;">
                        HawkEye View
                    </button>
                </div>
            </div>
        `);
        
        // Load incidents from HawkEye API
        function loadIncidents() {
            fetch('/api/v1/integrations/nnmi/incidents')
                .then(response => response.json())
                .then(data => {
                    const incidentList = container.select('#incident-list');
                    
                    if (!data || data.length === 0) {
                        incidentList.html('<div style="color: #888;">No active incidents</div>');
                        return;
                    }
                    
                    let html = '';
                    data.forEach(inc => {
                        const severityColor = {
                            'CRITICAL': '#f44336',
                            'MAJOR': '#ff9800',
                            'MINOR': '#ffeb3b',
                            'WARNING': '#2196f3',
                            'NORMAL': '#4caf50'
                        }[inc.severity] || '#888';
                        
                        html += `
                            <div style="margin-bottom: 10px; padding: 8px; background: #2d2d3a; border-radius: 4px; border-left: 4px solid ${severityColor};">
                                <div style="display: flex; justify-content: space-between;">
                                    <strong>${inc.name}</strong>
                                    <span style="color: ${severityColor};">${inc.severity}</span>
                                </div>
                                <div style="font-size: 12px; color: #aaa; margin-top: 5px;">
                                    ${inc.message || inc.formattedMessage || ''}
                                </div>
                                <div style="font-size: 11px; color: #888; margin-top: 5px;">
                                    ${inc.sourceNodeName || 'Unknown'} • ${new Date(inc.lastOccurrenceTime || inc.firstOccurrenceTime || Date.now()).toLocaleString()}
                                </div>
                            </div>
                        `;
                    });
                    
                    incidentList.html(html);
                })
                .catch(err => {
                    container.select('#incident-list')
                        .html(`<div style="color: #f44336;">Error loading incidents: ${err.message}</div>`);
                });
        }
        
        // Load incidents on init
        loadIncidents();
        
        // Handle refresh button
        container.select('#refresh-incidents').on('click', function() {
            loadIncidents();
        });
        
        // Handle HawkEye view button
        container.select('#hawkeye-details').on('click', function() {
            // Open HawkEye detailed view in new tab
            window.open('/hawkeye/nnmi-integration', '_blank');
        });
        
        // Register for real-time updates if data channel exists
        if (ctx.hasDataChannel) {
            ctx.onChange({
                callback: function(data) {
                    if (data && data.type === 'incident_update') {
                        loadIncidents(); // Refresh on update
                    }
                }
            });
        }
    },
    
    customProperty: [{
        id: 'bvd_refresh_interval',
        label: 'Refresh Interval (seconds)',
        type: 'number',
        default: 30
    }, {
        id: 'bvd_max_incidents',
        label: 'Max Incidents to Show',
        type: 'number',
        default: 10
    }],
    
    hasData: true,
    hasDataChannel: true
});
