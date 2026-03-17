import os
import re

files = ["alerts_config.html", "inventory.html", "live_dashboard.html", "polling_config.html"]

css_snippet = """
        .tree-parent {
            justify-content: space-between !important;
        }

        .tree-parent .tree-chevron {
            font-size: 0.7rem;
            transition: transform 0.2s;
            width: auto !important;
        }

        .tree-parent.open .tree-chevron {
            transform: rotate(180deg);
        }

        .tree-children {
            display: none;
            flex-direction: column;
            gap: 2px;
            background: rgba(0, 0, 0, 0.15);
            border-left: 2px solid rgba(139, 92, 246, 0.3);
            margin-left: 24px;
            padding-left: 4px;
            margin-top: 4px;
            margin-bottom: 4px;
        }

        .tree-parent.open + .tree-children {
            display: flex;
        }

        .child-item {
            padding: 8px 1rem !important;
            font-size: 0.8rem !important;
        }
"""

for fpath in files:
    with open(fpath, "r") as f:
        content = f.read()

    if ".tree-parent {" not in content:
        content = content.replace("</style>", css_snippet + "</style>")

    is_alerts_active = '"/alerts" class="nav-item active"' in content
    is_polling_active = '"/polling" class="nav-item active"' in content

    alerts_class = "nav-item child-item active" if is_alerts_active else "nav-item child-item"
    polling_class = "nav-item child-item active" if is_polling_active else "nav-item child-item"
    
    is_open = is_alerts_active or is_polling_active
    parent_open_class = " open" if is_open else ""
    active_wrench = " active" if is_open else ""

    replacement = f"""                <div class="nav-category" style="margin-top: 5px;">CONFIGURATION</div>
                <div class="nav-item tree-parent{parent_open_class}{active_wrench}" onclick="this.classList.toggle('open')">
                    <div style="display: flex; align-items: center; gap: 14px;"><i class="fa-solid fa-wrench"></i> Configuration</div>
                    <i class="fa-solid fa-chevron-down tree-chevron"></i>
                </div>
                <div class="tree-children">
                    <a href="/alerts" class="{alerts_class}">
                        <i class="fa-solid fa-folder"></i> Alerts Config
                    </a>
                    <a href="/polling" class="{polling_class}">
                        <i class="fa-solid fa-folder"></i> Polling Config
                    </a>
                </div>"""

    pattern = re.compile(r'<a href="/alerts".*?<a href="/polling"[^>]*>\s*<i class="fa-solid fa-clock-rotate-left"></i>\s*Polling Config\s*</a>', re.DOTALL)
    
    content = pattern.sub(replacement, content)
    
    with open(fpath, "w") as f:
        f.write(content)

print("Done")
