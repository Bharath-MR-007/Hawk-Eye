/**
 * Hawk-Eye UI Support Script
 * Handles: Theme toggling, Logout, and User display synchronization.
 */

const HawkUI = {
    init() {
        console.log("HawkUI: Initializing...");
        const steps = [
            { name: 'Theme', fn: () => this.initTheme() },
            { name: 'UserDisplay', fn: () => this.updateUserDisplay() },
            { name: 'RBAC', fn: () => this.applyRBAC() },
            { name: 'Nav', fn: () => this.setActiveNavItem() },
            { name: 'Cleanup', fn: () => this.cleanupURLParams() },
            { name: 'EventListeners', fn: () => this.setupEventListeners() }
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

    setActiveNavItem() {
        const path = window.location.pathname;
        const navItems = document.querySelectorAll('.nav-item');
        navItems.forEach(item => {
            const href = item.getAttribute('href');
            if (href === path || (path === '/' && href === '/dashboard')) {
                item.classList.add('active');
            } else {
                item.classList.remove('active');
            }
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
    }
};

// Auto-init helper
const initHawkUI = () => {
    console.log("Hawk-Eye UI initializing...");
    HawkUI.init();
};

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initHawkUI);
} else {
    initHawkUI();
}
