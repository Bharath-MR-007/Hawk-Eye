/**
 * Hawk-Eye UI Support Script
 * Handles: Theme toggling, Logout, and User display synchronization.
 */

const HawkUI = {
    init() {
        console.log("HawkUI: Initializing...");
        // Order matters: CSS first, sidebar second, theme third (theme button must exist)
        const steps = [
            { name: 'ThemeCSS',      fn: () => this.injectThemeLink() },
            { name: 'UserDisplay',   fn: () => this.updateUserDisplay() },
            { name: 'RBAC',          fn: () => this.applyRBAC() },
            { name: 'Sidebar',       fn: () => this.initSidebar() },
            { name: 'Theme',         fn: () => this.initTheme() },
            { name: 'Nav',           fn: () => this.setActiveNavItem() },
            { name: 'Cleanup',       fn: () => this.cleanupURLParams() },
            { name: 'EventListeners',fn: () => this.setupEventListeners() }
        ];

        steps.forEach(step => {
            try {
                step.fn();
                console.log(`HawkUI: ${step.name} initialized.`);
            } catch (err) {
                console.error(`HawkUI: Error initializing ${step.name}:`, err);
            }
        });
    },


    // ── Production theme CSS injection ────────────────────
    injectThemeLink() {
        if (document.getElementById('hawk-theme-css-link')) return;
        const link = document.createElement('link');
        link.id   = 'hawk-theme-css-link';
        link.rel  = 'stylesheet';
        link.href = '/scripts/hawk-theme.css';
        // Insert as first child of head so page styles can override
        document.head.insertBefore(link, document.head.firstChild);
    },

    // ── Sidebar injection ──────────────────────────────────
    initSidebar() {
        // Skip on login page
        const path = window.location.pathname;
        if (path === '/login') return;

        // Check if layout is already present (either via HTML or previous injection)
        const hasSidebar = document.querySelector('.main-sidebar') || document.getElementById('hawk-global-sidebar');
        if (hasSidebar) {
            console.log("HawkUI: Page already has a sidebar. Skipping layout injection.");
            // Theme button is handled by document-level delegation in initTheme() — do NOT bind here.
            return;
        }

        // ── CSS for injected layout ──
        const css = document.createElement('style');
        css.id = 'hawk-sidebar-css';
        css.textContent = `
            /* ── Hawk-Eye injected layout shell ── */
            html, body { height: 100%; overflow: hidden; margin: 0; padding: 0; }
            .hawk-app-window {
                display: flex;
                height: 100vh;
                width: 100vw;
                overflow: hidden;
                background: var(--he-bg-2, #080d18);
            }

            /* ── Global sidebar (injected) ── */
            #hawk-global-sidebar {
                width: 232px;
                min-width: 232px;
                background: var(--he-sidebar-bg, linear-gradient(175deg,#040a16,#060e22,#08132e));
                display: flex;
                flex-direction: column;
                flex-shrink: 0;
                z-index: 200;
                border-right: 1px solid var(--he-sidebar-border, rgba(0,212,255,0.08));
                box-shadow: var(--he-sidebar-shadow, 6px 0 40px rgba(0,0,0,0.6));
                transition: box-shadow 0.3s;
            }

            /* ── Sidebar header ── */
            .hawk-sidebar-header {
                height: 62px;
                display: flex;
                align-items: center;
                padding: 0 1.2rem;
                gap: 12px;
                background: rgba(0,0,0,0.30);
                border-bottom: 1px solid rgba(0,212,255,0.08);
                flex-shrink: 0;
            }
            .hawk-sidebar-logo-text { display: flex; flex-direction: column; line-height: 1.1; }
            .hawk-sidebar-logo-text .hawk-name {
                font-weight: 800; font-size: 1.08rem; color: #e8f4ff;
                letter-spacing: -0.015em;
            }
            .hawk-sidebar-logo-text .hawk-sub {
                font-size: 0.54rem; color: var(--he-accent,#00d4ff); font-weight: 700;
                text-transform: uppercase; letter-spacing: 2px; opacity: 0.85;
            }

            /* ── Sidebar nav ── */
            .hawk-sidebar-nav {
                flex: 1; overflow-y: auto; padding: 0.8rem 0;
                display: flex; flex-direction: column; gap: 1px;
                scrollbar-width: thin;
                scrollbar-color: rgba(0,212,255,0.10) transparent;
            }
            .hawk-nav-section {
                padding: 1rem 1rem 0.3rem;
                font-size: 0.58rem; font-weight: 800;
                text-transform: uppercase; letter-spacing: 0.14em;
                color: rgba(0,212,255,0.38);
            }
            .hawk-nav-item {
                display: flex; align-items: center;
                padding: 9px 1.2rem;
                color: rgba(172,210,255,0.72);
                text-decoration: none; font-size: 0.80rem; font-weight: 500;
                transition: all 0.16s ease;
                border-left: 3px solid transparent;
                gap: 11px; cursor: pointer;
                background: transparent;
                width: 100%; text-align: left; font-family: inherit;
            }
            .hawk-nav-item i {
                width: 16px; font-size: 0.88rem; text-align: center;
                color: rgba(0,212,255,0.40); transition: color 0.16s; flex-shrink: 0;
            }
            .hawk-nav-item:hover {
                background: rgba(0,212,255,0.09);
                color: #e0f4ff;
                border-left-color: rgba(0,212,255,0.50);
            }
            .hawk-nav-item:hover i { color: var(--he-accent,#00d4ff); }
            .hawk-nav-item.active {
                background: rgba(0,212,255,0.15);
                color: #c8ecff;
                border-left-color: var(--he-accent,#00d4ff);
                font-weight: 700;
            }
            .hawk-nav-item.active i { color: var(--he-accent,#00d4ff); }
            .hawk-tree-children {
                display: none; flex-direction: column;
                background: rgba(0,0,0,0.25);
                border-left: 2px solid rgba(0,212,255,0.18);
                margin-left: 22px;
            }
            .hawk-tree-parent.open + .hawk-tree-children { display: flex; }

            /* ── Page shell ── */
            .hawk-page-content {
                flex: 1; display: flex; flex-direction: column;
                min-width: 0; overflow: hidden; position: relative;
            }
            .hawk-page-scroll { flex: 1; overflow-y: auto; overflow-x: hidden; height: 100%; }

            /* ── Top bar ── */
            .hawk-top-bar {
                height: 54px;
                background: rgba(4,10,22,0.85);
                backdrop-filter: blur(14px);
                -webkit-backdrop-filter: blur(14px);
                border-bottom: 1px solid rgba(0,212,255,0.06);
                display: flex; align-items: center;
                justify-content: flex-end;
                padding: 0 1.5rem; gap: 10px; flex-shrink: 0;
            }
            body.light-theme .hawk-top-bar {
                background: rgba(240,244,250,0.92);
                border-bottom: 1px solid rgba(0,0,0,0.08);
            }

            /* ── Theme button ── */
            #hawk-theme-btn {
                background: rgba(0,212,255,0.07);
                border: 1px solid rgba(0,212,255,0.18);
                color: var(--he-accent,#00d4ff);
                width: 34px; height: 34px; border-radius: 8px;
                cursor: pointer; display: flex; align-items: center;
                justify-content: center; transition: all 0.2s; font-size: 0.9rem;
            }
            #hawk-theme-btn:hover {
                background: rgba(0,212,255,0.16);
                box-shadow: 0 0 12px rgba(0,212,255,0.22);
            }
            body.light-theme #hawk-theme-btn {
                background: rgba(0,50,100,0.08);
                border-color: rgba(0,80,150,0.20);
                color: #0060a8;
            }
        `;
        document.head.appendChild(css);

        // ── Build sidebar items ──
        const nav = [
            { section: 'Monitoring' },
            { label: 'Visual Dashboard',    icon: 'fa-gauge-high',     href: '/dashboard' },
            { label: 'Live Table',          icon: 'fa-table-list',     href: '/live_dashboard' },
            { label: 'Target Inventory',    icon: 'fa-list',           href: '/inventory' },
            { label: 'Tracepath',           icon: 'fa-route',          href: '/target_detail' },
            { label: 'Incident Browsing', icon: 'fa-magnifying-glass', href: null, children: [
                { label: 'All Incidents',   icon: 'fa-table-cells',    href: '/incidents' },
            ]},

            { section: 'Integrations' },
            { label: 'External Apps Links', icon: 'fa-link',           href: '/integrations' },
            { label: 'Integration Config',  icon: 'fa-gears',          href: '/integrations_config' },
            { section: 'Configuration' },
            { label: 'Alerts Config',       icon: 'fa-folder',         href: '/alerts' },
            { label: 'Polling Config',      icon: 'fa-folder',         href: '/polling' },
            { label: 'Endpoint Manager',    icon: 'fa-network-wired',  href: '/endpoints' },
            { label: 'User Configuration',  icon: 'fa-user-gear',      href: '/users_config' },
            { label: 'Export Snapshot',     icon: 'fa-file-export',    href: '#',                     fn: "if(typeof exportConfig==='function'){exportConfig();}else{alert('Export not available on this page.');}" },
            { label: 'Import Snapshot',     icon: 'fa-file-import',    href: '#',                     fn: "if(document.getElementById('importFile')){document.getElementById('importFile').click();}else{alert('Import not available on this page.');}" },
            { section: 'Operations' },
            { label: 'Troubleshooting',     icon: 'fa-microchip',      href: '/troubleshooting' },
            { label: 'Device Reachability', icon: 'fa-satellite-dish', href: '/device_reachability' },
        ];

        let navHtml = '';
        for (const item of nav) {
            if (item.section) { navHtml += `<div class="hawk-nav-section">${item.section}</div>`; continue; }
            if (item.children) {
                navHtml += `<div class="hawk-nav-item hawk-tree-parent" onclick="this.classList.toggle('open')"><div style="display:flex;align-items:center;gap:11px"><i class="fa-solid ${item.icon}"></i>${item.label}</div><i class="fa-solid fa-chevron-down hawk-tree-chevron" style="font-size:0.6rem; transition:transform 0.2s;"></i></div>`;
                navHtml += `<div class="hawk-tree-children" style="display:none; flex-direction:column; background:rgba(0,0,0,0.15); border-left:2px solid rgba(96,165,250,0.25); margin-left:22px;">${item.children.map(c => `<a href="${c.href}" class="hawk-nav-item" style="padding:7px 1rem; font-size:0.75rem;"><i class="fa-solid ${c.icon}"></i>${c.label}</a>`).join('')}</div>`;
            } else {
                const tag = item.fn ? 'button' : 'a';
                const hrefAttr = item.fn ? '' : `href="${item.href}"`;
                const onclickAttr = item.fn ? `onclick="${item.fn}"` : '';
                const idAttr = item.id ? `id="${item.id}"` : '';
                const iconStyle = item.style ? `style="${item.style}"` : '';
                navHtml += `<${tag} ${hrefAttr} ${onclickAttr} ${idAttr} class="hawk-nav-item"><i class="fa-solid ${item.icon}" ${iconStyle}></i>${item.label}</${tag}>`;
            }
        }

        const sidebar = document.createElement('aside');
        sidebar.id = 'hawk-global-sidebar';
        sidebar.innerHTML = `<div class="hawk-sidebar-header"><span style="font-size:1.6rem;line-height:1">🦅</span><div class="hawk-sidebar-logo-text"><span class="hawk-name">Hawk-Eye</span><span class="hawk-sub">T-Systems</span></div></div><nav class="hawk-sidebar-nav">${navHtml}</nav>`;

        const topBar = document.createElement('div');
        topBar.className = 'hawk-top-bar';
        topBar.innerHTML = `<button id="hawk-theme-btn" title="Toggle Dark / Light Mode"><i id="theme-toggle-icon" class="fa-solid fa-sun"></i></button>`;

        const existingChildren = Array.from(document.body.childNodes);
        const pageScroll = document.createElement('div');
        pageScroll.className = 'hawk-page-scroll';
        existingChildren.forEach(n => pageScroll.appendChild(n));

        const pageContent = document.createElement('div');
        pageContent.className = 'hawk-page-content';
        pageContent.appendChild(topBar);
        pageContent.appendChild(pageScroll);

        const appWindow = document.createElement('div');
        appWindow.className = 'hawk-app-window';
        appWindow.appendChild(sidebar);
        appWindow.appendChild(pageContent);

        document.body.appendChild(appWindow);
        // Theme button is handled by document-level delegation in initTheme() — do NOT bind here.
    },

    setActiveNavItem() {
        const path = window.location.pathname;
        // Handle both old (.nav-item) and new injected (.hawk-nav-item) items
        document.querySelectorAll('.hawk-nav-item[href], .nav-item[href]').forEach(item => {
            const href = item.getAttribute('href');
            const matches = href && (href === path || (path === '/' && href === '/dashboard'));
            item.classList.toggle('active', matches);
        });
    },


    cleanupURLParams() {
        const url = new URL(window.location.href);
        const params = url.searchParams;
        const toClean = ['export', 'add', 'import'];
        let needsCleanup = false;

        toClean.forEach(p => {
            if (params.get(p) === 'true') {
                needsCleanup = true;
            }
        });

        if (needsCleanup) {
            // Wait slightly longer than DOMContentLoaded to ensure other listeners fired
            setTimeout(() => {
                const freshUrl = new URL(window.location.href);
                toClean.forEach(p => freshUrl.searchParams.delete(p));
                window.history.replaceState({}, document.title, freshUrl.pathname + freshUrl.search);
                console.log("HawkUI: URL actions cleaned up.");
            }, 500);
        }
    },

    applyRBAC() {
        const role = this.getCookie('hawk_role');
        if (role === 'Operator') {
            console.log("Applying Operator restrictions...");

            // 1. Hide Management/Configuration action buttons
            const actionButtons = document.querySelectorAll('button.nav-item, .btn-primary, .btn-danger, #mainDeleteBtn, [onclick*="submit"], [onclick*="delete"], [onclick*="showAddUserModal"]');
            actionButtons.forEach(btn => {
                const text = (btn.innerText || btn.textContent || "").toLowerCase();
                if (text.includes('add') || text.includes('delete') || text.includes('import') || text.includes('export') || text.includes('create') || text.includes('update') || text.includes('save')) {
                    btn.style.setProperty('display', 'none', 'important');
                }
            });

            // 2. Disable Configuration links and add Lock icons
            const restrictedPaths = ['/alerts', '/polling', '/users_config', '/integrations_config'];
            const navLinks = document.querySelectorAll('a.nav-item');
            navLinks.forEach(link => {
                const href = link.getAttribute('href');
                if (href && restrictedPaths.some(path => href.includes(path))) {
                    link.style.opacity = '0.5';
                    link.style.pointerEvents = 'none';
                    link.style.cursor = 'not-allowed';
                    link.title = 'Access Denied: Read-Only User';

                    const icon = link.querySelector('i');
                    if (icon) icon.className = 'fa-solid fa-lock';
                }
            });

            // 3. Hide "Actions" column in all tables
            const tableHeaders = document.querySelectorAll('th');
            tableHeaders.forEach((th, index) => {
                if (th.textContent.toLowerCase().includes('action')) {
                    th.style.display = 'none';
                    const rows = th.closest('table').querySelectorAll('tr');
                    rows.forEach(row => {
                        const cell = row.cells[index];
                        if (cell) cell.style.display = 'none';
                    });
                }
            });
        }
    },

    initTheme() {
        // 1. Apply saved preference immediately (no flash)
        const saved = localStorage.getItem('hawk_theme') || 'dark';
        document.body.classList.toggle('light-theme', saved === 'light');
        this.updateThemeIcon();
        console.log(`HawkUI: Theme set to "${saved}"`);

        // 2. Wire ALL possible theme buttons directly (no event delegation layers)
        //    Run once now (for buttons already in DOM) and once after a tick
        //    (for buttons injected by initSidebar on pages without inline sidebar).
        const wireBtns = () => {
            document.querySelectorAll(
                '#theme-toggle-btn, #hawk-theme-btn, .theme-toggle-btn, [data-theme-toggle]'
            ).forEach(btn => {
                // Remove any old listener by cloning the node
                const fresh = btn.cloneNode(true);
                btn.parentNode.replaceChild(fresh, btn);
                fresh.addEventListener('click', (e) => {
                    e.preventDefault();
                    e.stopImmediatePropagation();
                    HawkUI.toggleTheme();
                });
            });
        };
        wireBtns();
        // Also wire after a microtask in case sidebar was just injected
        Promise.resolve().then(wireBtns);
    },

    updateThemeIcon() {
        const isLight = document.body.classList.contains('light-theme');
        const iconSel = [
            '#theme-toggle-icon',
            '#theme-toggle-btn i', '#theme-toggle-btn .fa-solid',
            '#hawk-theme-btn i',   '#hawk-theme-btn .fa-solid',
            '.theme-toggle-btn i', '[data-theme-toggle] i'
        ].join(', ');
        document.querySelectorAll(iconSel).forEach(icon => {
            icon.className = isLight ? 'fa-solid fa-moon' : 'fa-solid fa-sun';
        });
        document.querySelectorAll('#theme-toggle-btn, #hawk-theme-btn, .theme-toggle-btn').forEach(btn => {
            btn.setAttribute('title', isLight ? 'Switch to Dark Mode' : 'Switch to Light Mode');
        });
    },

    toggleTheme() {
        const isLight = document.body.classList.toggle('light-theme');
        localStorage.setItem('hawk_theme', isLight ? 'light' : 'dark');
        this.updateThemeIcon();
        console.log(`HawkUI: Theme toggled → ${isLight ? 'light' : 'dark'}`);
    },

    logout() {
        // Clear cookies
        document.cookie = "hawk_session=; path=/; max-age=0";
        document.cookie = "hawk_user=; path=/; max-age=0";
        // Redirect to logout endpoint (which handles backend cleanup)
        window.location.href = "/logout";
    },

    getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return decodeURIComponent(parts.pop().split(';').shift());
        return null;
    },

    updateUserDisplay() {
        const user = this.getCookie('hawk_user');
        const displayEl = document.getElementById('displayUser');
        if (displayEl && user) {
            displayEl.textContent = user;
        }
    },

    setupEventListeners() {
        // Logout buttons (still per-element since they don't conflict)
        document.querySelectorAll('#logout-btn, #logout-btn-sidebar').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                this.logout();
            });
        });
        // NOTE: Theme button is handled via document-level delegation in initTheme().
        // Do NOT add a second listener here — it would cause double-toggle.
    },

    initAIAssistant() {
        // Skip if already injected
        if (document.getElementById('ai-chat-widget')) return;

        // Styles
        const style = document.createElement('style');
        style.innerHTML = `
            #ai-chat-widget { position: fixed; bottom: 24px; right: 24px; z-index: 99999; font-family: 'Inter', sans-serif; }
            #ai-chat-toggle {
                width: 58px; height: 58px; border-radius: 50%;
                background: linear-gradient(135deg, #00d4ff 0%, #0095c8 60%, #006a9a 100%);
                color: #05080f; border: none;
                box-shadow: 0 4px 24px rgba(0,212,255,0.45);
                cursor: pointer; display: flex; align-items: center; justify-content: center;
                transition: transform 0.3s, box-shadow 0.3s; position: relative;
            }
            #ai-chat-toggle:hover { transform: scale(1.08); box-shadow: 0 6px 32px rgba(0,212,255,0.70); }
            #ai-chat-toggle .ai-pulse {
                position: absolute; top: 4px; right: 4px;
                width: 12px; height: 12px; border-radius: 50%;
                background: #10b981; border: 2px solid #0f1115;
                animation: ai-pulse-anim 2s infinite;
            }
            @keyframes ai-pulse-anim {
                0%,100% { transform: scale(1); opacity:1; }
                50% { transform: scale(1.3); opacity:0.7; }
            }
            #ai-chat-window {
                display: none; position: absolute; bottom: 72px; right: 0;
                width: 378px; height: 530px;
                background: rgba(5,10,22,0.97);
                border: 1px solid rgba(0,212,255,0.20);
                border-radius: 16px;
                box-shadow: 0 20px 60px rgba(0,0,0,0.7), 0 0 0 1px rgba(0,212,255,0.07);
                flex-direction: column; overflow: hidden;
                backdrop-filter: blur(24px); color: white;
            }
            .ai-header {
                padding: 14px 16px;
                background: linear-gradient(135deg, #004f6e 0%, #006a8a 50%, #0095c8 100%);
                color: white; font-weight: 700; font-size: 0.9rem;
                display: flex; justify-content: space-between; align-items: center;
                flex-shrink: 0;
                border-bottom: 1px solid rgba(0,212,255,0.15);
            }
            .ai-header-left { display: flex; align-items: center; gap: 10px; }
            .ai-status-dot { width: 8px; height: 8px; border-radius: 50%; background: #10b981; display: inline-block; }
            .ai-messages {
                flex: 1; overflow-y: auto; padding: 14px;
                display: flex; flex-direction: column; gap: 10px;
                scrollbar-width: thin; scrollbar-color: rgba(0,212,255,0.18) transparent;
            }
            .ai-messages::-webkit-scrollbar { width: 4px; }
            .ai-messages::-webkit-scrollbar-thumb { background: rgba(0,212,255,0.25); border-radius: 2px; }
            .ai-quick-actions { padding: 0 14px 10px; display: flex; gap: 6px; flex-wrap: wrap; flex-shrink: 0; }
            .ai-quick-btn {
                font-size: 0.7rem; padding: 4px 10px; border-radius: 20px;
                background: rgba(0,212,255,0.08); color: #6ee8ff;
                border: 1px solid rgba(0,212,255,0.22); cursor: pointer;
                transition: background 0.2s;
            }
            .ai-quick-btn:hover { background: rgba(0,212,255,0.18); color: #b8f5ff; }
            .ai-input-area {
                padding: 12px 14px; border-top: 1px solid rgba(0,212,255,0.08);
                background: rgba(0,0,0,0.25); display: flex; gap: 8px;
                align-items: flex-end; flex-shrink: 0;
            }
            .ai-input {
                flex: 1; background: rgba(0,212,255,0.04);
                border: 1px solid rgba(0,212,255,0.14); border-radius: 10px;
                padding: 9px 12px; color: #e0f4ff; font-size: 0.82rem;
                outline: none; resize: none; height: 38px; max-height: 90px;
                transition: border-color 0.2s, box-shadow 0.2s; font-family: inherit;
            }
            .ai-input::placeholder { color: rgba(160,210,255,0.40); }
            .ai-input:focus { border-color: rgba(0,212,255,0.45); box-shadow: 0 0 0 3px rgba(0,212,255,0.08); }
            .ai-send {
                background: linear-gradient(135deg,#00d4ff,#0095c8); color: #04080f; border: none;
                border-radius: 10px; width: 36px; height: 36px;
                cursor: pointer; display: flex; align-items: center;
                justify-content: center; flex-shrink: 0; transition: all 0.2s;
                font-weight: 700;
            }
            .ai-send:hover { filter: brightness(1.15); box-shadow: 0 4px 16px rgba(0,212,255,0.35); }
            .msg-user {
                align-self: flex-end;
                background: linear-gradient(135deg, #004f6e, #0085b0);
                color: #e0f8ff; padding: 9px 13px; border-radius: 14px 14px 2px 14px;
                font-size: 0.82rem; max-width: 82%; line-height: 1.45;
                box-shadow: 0 2px 10px rgba(0,212,255,0.22);
            }
            .msg-ai {
                align-self: flex-start;
                background: rgba(0,212,255,0.05);
                color: #d4eeff; padding: 9px 13px;
                border-radius: 14px 14px 14px 2px;
                font-size: 0.82rem; max-width: 85%; line-height: 1.5;
                border: 1px solid rgba(0,212,255,0.10);
            }
            .msg-ai-icon { font-size: 1rem; margin-right: 6px; vertical-align: middle; }
            .ai-typing { display: flex; align-items: center; gap: 8px; color: #6ec0d4; font-size: 0.78rem; padding: 4px 0; }
            .ai-typing-dots span {
                display: inline-block; width: 5px; height: 5px;
                border-radius: 50%; background: var(--he-accent,#00d4ff); margin: 0 1px;
                animation: ai-dot 1.4s infinite both;
            }
            .ai-typing-dots span:nth-child(2) { animation-delay: 0.2s; }
            .ai-typing-dots span:nth-child(3) { animation-delay: 0.4s; }
            @keyframes ai-dot { 0%,80%,100%{transform:scale(0)} 40%{transform:scale(1)} }
        `;
        document.head.appendChild(style);

        // Widget HTML
        const widget = document.createElement('div');
        widget.id = 'ai-chat-widget';
        widget.innerHTML = `
            <button id="ai-chat-toggle" title="Observability AI Assistant">
                <i class="fa-solid fa-robot" style="font-size:1.4rem;"></i>
                <span class="ai-pulse"></span>
            </button>
            <div id="ai-chat-window">
                <div class="ai-header">
                    <div class="ai-header-left">
                        <span class="ai-status-dot"></span>
                        <span>🦅 Hawk-Eye AI</span>
                    </div>
                    <i class="fa-solid fa-xmark" id="ai-chat-close" style="cursor:pointer; opacity:0.8;"></i>
                </div>
                <div id="ai-chat-messages" class="ai-messages">
                    <div class="msg-ai">
                        <span class="msg-ai-icon">🤖</span>Hi! I'm your Hawk-Eye Observability AI.<br><br>
                        I can help you analyze system health, diagnose latency issues, decode alert patterns, or explain any metric.<br><br>
                        Ask anything — or pick a quick action below.
                    </div>
                </div>
                <div class="ai-quick-actions">
                    <button class="ai-quick-btn" data-q="What targets are critical right now?">🔴 Critical targets?</button>
                    <button class="ai-quick-btn" data-q="Why is DNS resolution failing?">🌐 DNS issues?</button>
                    <button class="ai-quick-btn" data-q="Explain high latency on amazon.com">⚡ High latency?</button>
                    <button class="ai-quick-btn" data-q="Which SSL certs are expiring soon?">🔒 SSL expiry?</button>
                </div>
                <div class="ai-input-area">
                    <textarea id="ai-chat-input" class="ai-input" placeholder="Ask about system health, alerts, latency..." rows="1"></textarea>
                    <button id="ai-chat-send" class="ai-send"><i class="fa-solid fa-paper-plane"></i></button>
                </div>
            </div>
        `;
        document.body.appendChild(widget);

        // Logic
        const aiWindow = document.getElementById('ai-chat-window');
        const toggle = document.getElementById('ai-chat-toggle');
        const closeBtn = document.getElementById('ai-chat-close');
        const input = document.getElementById('ai-chat-input');
        const send = document.getElementById('ai-chat-send');
        const msgs = document.getElementById('ai-chat-messages');

        toggle.onclick = () => {
            const isOpen = aiWindow.style.display === 'flex';
            aiWindow.style.display = isOpen ? 'none' : 'flex';
            toggle.style.transform = isOpen ? 'scale(1)' : 'scale(0.92) rotate(15deg)';
        };
        closeBtn.onclick = () => {
            aiWindow.style.display = 'none';
            toggle.style.transform = 'scale(1)';
        };

        document.querySelectorAll('.ai-quick-btn').forEach(btn => {
            btn.onclick = () => {
                input.value = btn.dataset.q;
                handleSend();
            };
        });

        const addMsg = (html, isUser = false) => {
            const div = document.createElement('div');
            div.className = isUser ? 'msg-user' : 'msg-ai';
            div.innerHTML = isUser ? html : `<span class="msg-ai-icon">🤖</span>${html}`;
            msgs.appendChild(div);
            msgs.scrollTop = msgs.scrollHeight;
            return div;
        };

        const handleSend = async () => {
            const text = input.value.trim();
            if (!text) return;
            addMsg(text, true);
            input.value = '';

            // Typing indicator
            const typingDiv = document.createElement('div');
            typingDiv.className = 'ai-typing';
            typingDiv.innerHTML = `<div class="ai-typing-dots"><span></span><span></span><span></span></div><span>Analyzing…</span>`;
            msgs.appendChild(typingDiv);
            msgs.scrollTop = msgs.scrollHeight;

            // Context enrichment
            let context = '';
            const incMatch = text.match(/inc-[a-z0-9]{4,12}/i);
            if (incMatch) {
                const uuidMap = JSON.parse(localStorage.getItem('hawkeye_uuid_map') || '{}');
                const inc = uuidMap[incMatch[0].toLowerCase()];
                if (inc) context += `\n[CRITICAL INCIDENT DATA: ${JSON.stringify(inc)}]`;
            }

            try {
                const res = await fetch('http://localhost:8000/chat', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ message: text, context })
                });
                const data = await res.json();
                typingDiv.remove();
                addMsg(data.response || "I couldn't generate a response.");
            } catch {
                typingDiv.remove();
                addMsg('AI Assistant is currently offline.<br><small style="color:#94a3b8;">Start <code>web_api.py</code> on port 8000 to enable AI features.</small>');
            }
        };

        send.onclick = handleSend;
        input.addEventListener('keypress', e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); } });
    }
};

// ── Apply saved theme IMMEDIATELY on script parse (prevents dark flash on light mode) ──
(function() {
    try {
        const t = localStorage.getItem('hawk_theme');
        if (t === 'light') document.documentElement.classList.add('light-theme');
    } catch(e) {}
})();

// ── Expose globally so inline onclick="HawkUI.toggleTheme()" still works ──
window.HawkUI = HawkUI;

// ── Auto-init on DOM ready ──
const initHawkUI = () => {
    console.log("Hawk-Eye UI initializing...");
    HawkUI.init();
    HawkUI.initAIAssistant();
};

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initHawkUI);
} else {
    initHawkUI();
}
