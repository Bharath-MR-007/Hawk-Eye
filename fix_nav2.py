import re
import os

files = ["alerts_config.html", "inventory.html", "live_dashboard.html", "polling_config.html"]

for fpath in files:
    if os.path.exists(fpath):
        with open(fpath, "r") as f:
            content = f.read()

        # 1. Remove exactly the <div class="nav-category" ...>CONFIGURATION</div> inside <nav>
        content = re.sub(r'<div class="nav-category"[^>]*>CONFIGURATION</div>\s*', '', content, flags=re.IGNORECASE)
        # 2. Also remove any `<div class="nav-category">Configuration</div>`
        content = re.sub(r'<div class="nav-category"[^>]*>Configuration</div>\s*', '', content, flags=re.IGNORECASE)

        # 3. Locate the Export and Import Snapshot buttons manually
        # In live_dashboard.html, they look like:
        # <button class="nav-item" onclick="exportConfig()">
        #     <i class="fa-solid fa-file-export"></i> Export Snapshot
        # </button>
        export_btn = ''
        import_btn = ''
        
        # We need to find the <button> tags that contain Export Snapshot and Import Snapshot.
        # We can extract them exactly.
        idx_export = content.find('<button class="nav-item" onclick="exportConfig()">')
        if idx_export != -1:
            end_export = content.find("</button>", idx_export) + len("</button>")
            export_btn = content[idx_export:end_export] + "\n"
            content = content[:idx_export] + content[end_export:]
            
        # Refind because index changed
        idx_import = content.find('<button class="nav-item" onclick="document.getElementById(\'importFile\').click()">')
        if idx_import != -1:
            end_import = content.find("</button>", idx_import) + len("</button>")
            import_btn = content[idx_import:end_import] + "\n"
            content = content[:idx_import] + content[end_import:]

        # Now replace 'nav-item' with 'nav-item child-item' and align text inside the buttons.
        export_btn = export_btn.replace('class="nav-item"', 'class="nav-item child-item" style="text-align: left;"')
        import_btn = import_btn.replace('class="nav-item"', 'class="nav-item child-item" style="text-align: left;"')

        # Find the tree-children closing tag
        idx_end_tree = content.find("</div>\n                <div class=\"nav-category\">Management</div>")
        if idx_end_tree == -1:
            # Fallback if Management doesn't exist right after
            idx_end_tree = content.find("</div>\n            </nav>")
            
        if idx_end_tree != -1:
            # Inject
            content = content[:idx_end_tree] + "    " + export_btn + "    " + import_btn + content[idx_end_tree:]

        with open(fpath, "w") as f:
            f.write(content)
print("Done")
