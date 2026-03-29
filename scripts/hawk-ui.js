/**
 * Hawk-Eye UI Support Script
 * Handles: Theme toggling, Logout, and User display synchronization.
 */

const HawkUI = {
    init() {
        console.log("HawkUI: Initializing...");
        const steps = [
            { name: 'Theme',         fn: () => this.initTheme() },
            { name: 'UserDisplay',   fn: () => this.updateUserDisplay() },
            { name: 'RBAC',          fn: () => this.applyRBAC() },
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


    // ── Sidebar injection ──────────────────────────────────
    initSidebar() {
        // Skip on login page
        const path = window.location.pathname;
        if (path === '/login') return;

        // Check if layout is already present (either via HTML or previous injection)
        const hasSidebar = document.querySelector('.main-sidebar') || document.getElementById('hawk-global-sidebar');
        if (hasSidebar) {
            console.log("HawkUI: Page already has a sidebar. Skipping layout injection.");
            // Just wire the existing theme button if it exists
            const themeBtn = document.getElementById('theme-toggle-btn');
            if (themeBtn) {
                themeBtn.onclick = (e) => { e.preventDefault(); this.toggleTheme(); };
            }
            return;
        }

        // ── CSS for injected layout ──
        const css = document.createElement('style');
        css.id = 'hawk-sidebar-css';
        css.textContent = `
            html, body { height: 100%; overflow: hidden; margin: 0; padding: 0; }
            .hawk-app-window {
                display: flex;
                height: 100vh;
                width: 100vw;
                overflow: hidden;
            }
            #hawk-global-sidebar {
                width: 230px;
                min-width: 230px;
                background: linear-gradient(175deg, #0b1a2e, #0f2744, #122f52);
                display: flex;
                flex-direction: column;
                flex-shrink: 0;
                z-index: 200;
                border-right: 1px solid rgba(255,255,255,0.08);
                transition: background 0.3s;
            }
            body.light-theme #hawk-global-sidebar { background: linear-gradient(175deg, #1a3a6c, #1e4487, #1d4ed8); }
            .hawk-sidebar-header {
                height: 62px;
                display: flex;
                align-items: center;
                padding: 0 1.2rem;
                gap: 12px;
                border-bottom: 1px solid rgba(255,255,255,0.1);
                flex-shrink: 0;
            }
            .hawk-sidebar-logo-text { display: flex; flex-direction: column; line-height: 1.1; }
            .hawk-sidebar-logo-text .hawk-name { font-weight: 800; font-size: 1.05rem; color: #e0f0ff; letter-spacing: -0.01em; }
            .hawk-sidebar-logo-text .hawk-sub { font-size: 0.58rem; color: #ff6ef7; font-weight: 700; text-transform: uppercase; letter-spacing: 1.5px; }
            .hawk-sidebar-nav { flex: 1; overflow-y: auto; padding: 1rem 0; display: flex; flex-direction: column; gap: 1px; scrollbar-width: thin; scrollbar-color: rgba(255,255,255,0.1) transparent; }
            .hawk-nav-section { padding: 0.9rem 1rem 0.3rem; font-size: 0.6rem; font-weight: 800; text-transform: uppercase; letter-spacing: 0.1em; color: rgba(255,255,255,0.4); }
            .hawk-nav-item {
                display: flex; align-items: center; padding: 9px 1.2rem; color: rgba(200,220,255,0.75);
                text-decoration: none; font-size: 0.8rem; font-weight: 600; transition: all 0.15s; border-left: 3px solid transparent; gap: 11px; cursor: pointer; background: transparent; width: 100%; text-align: left; font-family: inherit;
            }
            .hawk-nav-item i { width: 16px; font-size: 0.9rem; text-align: center; color: rgba(160,200,255,0.6); transition: color 0.15s; flex-shrink: 0; }
            .hawk-nav-item:hover { background: rgba(255,255,255,0.07); color: #e0f0ff; border-left-color: rgba(96,165,250,0.5); }
            .hawk-nav-item:hover i { color: #93c5fd; }
            .hawk-nav-item.active { background: rgba(59,130,246,0.2); color: #e0f0ff; border-left-color: #60a5fa; font-weight: 700; }
            .hawk-page-content { flex: 1; display: flex; flex-direction: column; min-width: 0; overflow: hidden; position: relative; }
            .hawk-page-scroll { flex: 1; overflow-y: auto; overflow-x: hidden; height: 100%; }
            .hawk-top-bar { height: 56px; background: rgba(0,0,0,0.15); border-bottom: 1px solid rgba(255,255,255,0.06); display: flex; align-items: center; justify-content: flex-end; padding: 0 1.5rem; gap: 10px; flex-shrink: 0; }
            body.light-theme .hawk-top-bar { background: rgba(255,255,255,0.6); border-bottom: 1px solid rgba(0,0,0,0.08); }
            #hawk-theme-btn { background: rgba(255,255,255,0.08); border: 1px solid rgba(255,255,255,0.12); color: rgba(200,220,255,0.8); width: 34px; height: 34px; border-radius: 8px; cursor: pointer; display: flex; align-items: center; justify-content: center; transition: all 0.2s; font-size: 0.9rem; }
            body.light-theme #hawk-theme-btn { background: rgba(30,64,128,0.1); border-color: rgba(30,64,128,0.2); color: #1e4080; }
        `;
        document.head.appendChild(css);

        // ── Build sidebar items ──
        const nav = [
            { section: 'Monitoring' },
            { label: 'Visual Dashboard',    icon: 'fa-gauge-high',     href: '/dashboard' },
            { label: 'Live Table',          icon: 'fa-table-list',     href: '/live_dashboard' },
            { label: 'Target Inventory',    icon: 'fa-list',           href: '/inventory' },
            { label: 'Incident Browsing', icon: 'fa-magnifying-glass', href: null, children: [
                { label: 'All Incidents',   icon: 'fa-table-cells',    href: '/incidents' },
            ]},
            { section: 'Management' },
            { label: 'Add New Target',      icon: 'fa-plus',           href: '/dashboard?add=true',           style: 'color:#60a5fa' },
            { label: 'Delete Target',       icon: 'fa-trash',          href: '/live_dashboard',       style: 'color:#f87171', id: 'mainDeleteBtn' },
            { section: 'Integrations' },
            { label: 'External Apps Links', icon: 'fa-link',           href: '/integrations' },
            { label: 'Integration Config',  icon: 'fa-gears',          href: '/integrations_config' },
            { section: 'Configuration' },
            { label: 'Alerts Config',       icon: 'fa-folder',         href: '/alerts' },
            { label: 'Polling Config',      icon: 'fa-folder',         href: '/polling' },
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
        topBar.querySelector('#hawk-theme-btn').addEventListener('click', () => this.toggleTheme());
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
        const savedTheme = localStorage.getItem('hawk_theme') || 'dark';
        if (savedTheme === 'light') {
            document.body.classList.add('light-theme');
        } else {
            document.body.classList.remove('light-theme');
        }
        
        // Update icon on init
        const icon = document.querySelector('#theme-toggle-icon');
        if (icon) {
            icon.className = savedTheme === 'light' ? 'fa-solid fa-moon' : 'fa-solid fa-sun';
        }
        console.log(`HawkUI: Theme initialized to ${savedTheme}`);
    },

    toggleTheme() {
        const isLight = document.body.classList.toggle('light-theme');
        localStorage.setItem('hawk_theme', isLight ? 'light' : 'dark');

        const icon = document.querySelector('#theme-toggle-icon');
        if (icon) {
            icon.className = isLight ? 'fa-solid fa-moon' : 'fa-solid fa-sun';
        }
        console.log(`HawkUI: Theme toggled. IsLight: ${isLight}`);
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
        // Any global listeners can go here
        const logoutBtns = document.querySelectorAll('#logout-btn, #logout-btn-sidebar');
        logoutBtns.forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                this.logout();
            });
        });

        const themeBtn = document.getElementById('theme-toggle-btn');
        if (themeBtn) {
            themeBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.toggleTheme();
            });
        }
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
                background: linear-gradient(135deg, #8b5cf6, #6d28d9);
                color: white; border: none;
                box-shadow: 0 4px 20px rgba(139,92,246,0.5);
                cursor: pointer; display: flex; align-items: center; justify-content: center;
                transition: transform 0.3s, box-shadow 0.3s; position: relative;
            }
            #ai-chat-toggle:hover { transform: scale(1.08); box-shadow: 0 6px 28px rgba(139,92,246,0.7); }
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
                width: 370px; height: 520px;
                background: rgba(22,25,35,0.97);
                border: 1px solid rgba(139,92,246,0.3);
                border-radius: 16px;
                box-shadow: 0 20px 60px rgba(0,0,0,0.6), 0 0 0 1px rgba(139,92,246,0.1);
                flex-direction: column; overflow: hidden;
                backdrop-filter: blur(20px); color: white;
            }
            .ai-header {
                padding: 14px 16px;
                background: linear-gradient(135deg, #8b5cf6 0%, #6d28d9 100%);
                color: white; font-weight: 700; font-size: 0.9rem;
                display: flex; justify-content: space-between; align-items: center;
                flex-shrink: 0;
            }
            .ai-header-left { display: flex; align-items: center; gap: 10px; }
            .ai-status-dot { width: 8px; height: 8px; border-radius: 50%; background: #10b981; display: inline-block; }
            .ai-messages {
                flex: 1; overflow-y: auto; padding: 14px;
                display: flex; flex-direction: column; gap: 10px;
                scrollbar-width: thin; scrollbar-color: rgba(139,92,246,0.3) transparent;
            }
            .ai-messages::-webkit-scrollbar { width: 4px; }
            .ai-messages::-webkit-scrollbar-thumb { background: rgba(139,92,246,0.4); border-radius: 2px; }
            .ai-quick-actions { padding: 0 14px 10px; display: flex; gap: 6px; flex-wrap: wrap; flex-shrink: 0; }
            .ai-quick-btn {
                font-size: 0.7rem; padding: 4px 10px; border-radius: 20px;
                background: rgba(139,92,246,0.15); color: #c4b5fd;
                border: 1px solid rgba(139,92,246,0.3); cursor: pointer;
                transition: background 0.2s;
            }
            .ai-quick-btn:hover { background: rgba(139,92,246,0.3); }
            .ai-input-area {
                padding: 12px 14px; border-top: 1px solid rgba(255,255,255,0.06);
                background: rgba(0,0,0,0.2); display: flex; gap: 8px;
                align-items: flex-end; flex-shrink: 0;
            }
            .ai-input {
                flex: 1; background: rgba(255,255,255,0.05);
                border: 1px solid rgba(255,255,255,0.1); border-radius: 10px;
                padding: 9px 12px; color: white; font-size: 0.82rem;
                outline: none; resize: none; height: 38px; max-height: 90px;
                transition: border-color 0.2s; font-family: inherit;
            }
            .ai-input:focus { border-color: rgba(139,92,246,0.5); }
            .ai-send {
                background: #8b5cf6; color: white; border: none;
                border-radius: 10px; width: 36px; height: 36px;
                cursor: pointer; display: flex; align-items: center;
                justify-content: center; flex-shrink: 0; transition: background 0.2s;
            }
            .ai-send:hover { background: #7c3aed; }
            .msg-user {
                align-self: flex-end;
                background: linear-gradient(135deg, #8b5cf6, #7c3aed);
                color: white; padding: 9px 13px; border-radius: 14px 14px 2px 14px;
                font-size: 0.82rem; max-width: 82%; line-height: 1.45;
                box-shadow: 0 2px 8px rgba(139,92,246,0.3);
            }
            .msg-ai {
                align-self: flex-start;
                background: rgba(255,255,255,0.06);
                color: #e2e8f0; padding: 9px 13px;
                border-radius: 14px 14px 14px 2px;
                font-size: 0.82rem; max-width: 85%; line-height: 1.5;
                border: 1px solid rgba(255,255,255,0.07);
            }
            .msg-ai-icon { font-size: 1rem; margin-right: 6px; vertical-align: middle; }
            .ai-typing { display: flex; align-items: center; gap: 8px; color: #94a3b8; font-size: 0.78rem; padding: 4px 0; }
            .ai-typing-dots span {
                display: inline-block; width: 5px; height: 5px;
                border-radius: 50%; background: #8b5cf6; margin: 0 1px;
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

// Auto-init helper
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
