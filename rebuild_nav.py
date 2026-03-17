import re
import os

files = [
    "alerts_config.html", "inventory.html", "live_dashboard.html", "polling_config.html", 
    "incidents.html", "integrations.html", "integrations_config.html", "integrations_guide.html",
    "capabilities.html", "installation.html", "requirements.html", "architecture.html", 
    "usermanual.html", "admindoc.html", "about.html", "login.html", "troubleshooting.html", 
    "device_reachability.html", "users_config.html", "target_detail.html"
]

def generate_header():
    return """            <div class="main-sidebar-header">
                <span style="font-size: 1.5rem;">🦅</span>
                <div style="display: flex; flex-direction: column; line-height: 1.1;">
                    <span style="font-weight: 700; font-size: 1.25rem;">Hawk-Eye</span>
                    <span
                        style="font-size: 0.65rem; color: #ff00ff; font-weight: 700; text-transform: uppercase; letter-spacing: 1px;">T-Systems</span>
                </div>
            </div>"""

def generate_nav(active_page):
    return f"""            <nav class="main-sidebar-nav">
                <div class="nav-category">Monitoring</div>
                <a href="/dashboard" class="nav-item{' active' if active_page == 'dashboard' else ''}">
                    <i class="fa-solid fa-gauge-high"></i> Dashboard
                </a>
                <a href="/inventory" class="nav-item{' active' if active_page == 'inventory' else ''}">
                    <i class="fa-solid fa-list"></i> Target Inventory
                </a>
                <a href="/incidents" class="nav-item{' active' if active_page == 'incidents' else ''}">
                    <i class="fa-solid fa-magnifying-glass"></i> All Incidents
                </a>

                <div class="nav-category">Management</div>
                <button class="nav-item" onclick="if(document.getElementById('addTargetModal')) {{ document.getElementById('addTargetModal').style.display='flex'; }} else {{ window.location.href='/dashboard?add=true'; }}">
                    <i class="fa-solid fa-plus" style="color: var(--info-color);"></i> Add New Target
                </button>
                <button class="nav-item" id="mainDeleteBtn" onclick="if(typeof deleteSelectedTargets === 'function') {{ deleteSelectedTargets(); }} else {{ window.location.href='/dashboard'; }}">
                    <i class="fa-solid fa-trash" style="color: var(--error-color);"></i> Delete Target
                </button>

                <div class="nav-category" style="margin-top: 15px;">Integrations</div>
                <a href="/integrations" class="nav-item{' active' if active_page == 'integrations' else ''}">
                    <i class="fa-solid fa-link"></i> External Apps Links
                </a>
                <a href="/integrations_config" class="nav-item{' active' if active_page == 'integrations_config' else ''}">
                    <i class="fa-solid fa-gears"></i> Integration Config
                </a>

                <div class="nav-category" style="margin-top: 15px;">Configuration</div>
                <a href="/alerts" class="nav-item{' active' if active_page == 'alerts' else ''}">
                    <i class="fa-solid fa-folder"></i> Alerts Config
                </a>
                <a href="/polling" class="nav-item{' active' if active_page == 'polling' else ''}">
                    <i class="fa-solid fa-folder"></i> Polling Config
                </a>
                <a href="/users_config" class="nav-item{' active' if active_page == 'users_config' else ''}">
                    <i class="fa-solid fa-user-gear"></i> User Configuration
                </a>
                <button class="nav-item" onclick="if(typeof exportConfig === 'function') {{ exportConfig(); }} else {{ window.location.href='/dashboard?export=true'; }}">
                    <i class="fa-solid fa-file-export"></i> Export Snapshot
                </button>
                <button class="nav-item" onclick="if(document.getElementById('importFile')) {{ document.getElementById('importFile').click(); }} else {{ window.location.href='/dashboard?import=true'; }}">
                    <i class="fa-solid fa-file-import"></i> Import Snapshot
                </button>

                <div class="nav-category" style="margin-top: 15px;">Operations</div>
                <a href="/troubleshooting" class="nav-item{' active' if active_page == 'troubleshooting' else ''}">
                    <i class="fa-solid fa-microchip"></i> Troubleshooting
                </a>
                <a href="/device_reachability" class="nav-item{' active' if active_page == 'device_reachability' else ''}">
                    <i class="fa-solid fa-satellite-dish"></i> Device Reachability
                </a>
            </nav>"""

def generate_help_menu():
    return """                        <div class="help-dropdown-content">
                            <a href="/installation.html"><i class="fa-solid fa-book"></i> Installation Procedure</a>
                            <a href="/requirements.html"><i class="fa-solid fa-list-check"></i> Software Requirements</a>
                            <a href="/capabilities.html"><i class="fa-solid fa-bolt"></i> Capabilities of Hawk-Eye</a>
                            <a href="/integrations_guide.html"><i class="fa-solid fa-layer-group"></i> Integration Guide</a>
                            <a href="/architecture.html"><i class="fa-solid fa-diagram-project"></i> Product Architecture</a>
                            <a href="/usermanual.html"><i class="fa-solid fa-file-lines"></i> User Manual</a>
                            <a href="/admindoc.html"><i class="fa-solid fa-user-shield"></i> Admin Document</a>
                            <a href="/about.html"><i class="fa-solid fa-circle-info"></i> About - Overview</a>
                        </div>"""

for fpath in files:
    if not os.path.exists(fpath):
        continue
    with open(fpath, "r") as f:
        content = f.read()

    # Determine active page
    active_page = ""
    if "dashboard" in fpath or "live" in fpath: active_page = "dashboard"
    elif "inventory" in fpath: active_page = "inventory"
    elif "incidents" in fpath: active_page = "incidents"
    elif "alerts" in fpath: active_page = "alerts"
    elif "polling" in fpath: active_page = "polling"
    elif "users_config" in fpath: active_page = "users_config"
    elif "device_reachability" in fpath: active_page = "device_reachability"
    elif "troubleshooting" in fpath: active_page = "troubleshooting"
    elif "integrations_config" in fpath: active_page = "integrations_config"
    elif "integrations" in fpath: active_page = "integrations"

    # Robust Header Replacement
    header_regex = r'<div class="main-sidebar-header">.*?(?=\s*<nav)'
    if re.search(header_regex, content, flags=re.DOTALL):
        content = re.sub(header_regex, generate_header() + "\n", content, count=1, flags=re.DOTALL)
    
    # Replace navigation block
    nav_pattern = r'<nav class="main-sidebar-nav">.*?</nav>'
    if re.search(nav_pattern, content, flags=re.DOTALL):
        content = re.sub(nav_pattern, generate_nav(active_page), content, flags=re.DOTALL)
    
    # Replace help menu block
    help_pattern = r'<div class="help-dropdown-content">.*?</div>'
    if re.search(help_pattern, content, flags=re.DOTALL):
        content = re.sub(help_pattern, generate_help_menu(), content, flags=re.DOTALL)
    
    # Revert Hawk-Eye branding
    content = re.sub(r'Hawk-Eye(?!\.log)', 'Hawk-Eye', content)
    
    # Hover Bridge CSS
    css_fix = """
        .help-dropdown-content::before {
            content: '';
            position: absolute;
            top: -20px;
            left: 0;
            width: 100%;
            height: 25px;
            background: transparent;
        }"""
    
    if ".help-dropdown-content::before" in content:
        content = re.sub(r'\.help-dropdown-content::before\s*\{.*?\}', css_fix, content, flags=re.DOTALL)
    else:
        content = content.replace("</style>", css_fix + "\n    </style>")

    with open(fpath, "w") as f:
        f.write(content)

print("Synchronized all pages specifically for Device Reachability.")
