import re

files = ["alerts_config.html", "inventory.html", "live_dashboard.html", "polling_config.html"]

for fpath in files:
    with open(fpath, "r") as f:
        content = f.read()

    # 1. Strip out the CONFIGURATION and Configuration nav-categories entirely
    content = re.sub(r'<div class="nav-category"[^>]*>CONFIGURATION</div>\s*', '', content)
    content = re.sub(r'<div class="nav-category"[^>]*>Configuration</div>\s*', '', content)

    # 2. Add Export Snapshot and Import Snapshot items properly formatted for inside a tree-children element.
    # Make sure we remove them from their original locations if they exist
    export_pattern = r'<button class="nav-item".*?<i class="fa-solid fa-file-export"></i>\s*Export Snapshot\s*</button>\s*'
    import_pattern = r'<button class="nav-item".*?<i class="fa-solid fa-file-import"></i>\s*Import Snapshot\s*</button>\s*'
    
    # Store originals to ensure we copy their JS safely
    export_btn = ''
    import_btn = ''
    
    export_match = re.search(export_pattern, content, flags=re.DOTALL)
    import_match = re.search(import_pattern, content, flags=re.DOTALL)
    
    if export_match:
        export_btn = export_match.group(0).replace('class="nav-item"', 'class="nav-item child-item"')
        content = content.replace(export_match.group(0), "")
    else:
        # Default placeholder if missing from the HTML file
        export_btn = """<button class="nav-item child-item" onclick="exportConfig()" style="text-align: left;"><i class="fa-solid fa-file-export"></i> Export Snapshot</button>
                    """
        
    if import_match:
        import_btn = import_match.group(0).replace('class="nav-item"', 'class="nav-item child-item"')
        content = content.replace(import_match.group(0), "")
    else:
        import_btn = """<button class="nav-item child-item" onclick="document.getElementById('importFile').click()" style="text-align: left;"><i class="fa-solid fa-file-import"></i> Import Snapshot</button>
                    """

    # 3. Inject them back into the tree-children component.
    # Find the closing tag of tree-children
    tree_content_end = r'(<div class="tree-children">.*?)(</div>)'
    
    def replace_tree(m):
        # Prevent double adding
        inner_html = m.group(1)
        if "Export Snapshot" not in inner_html:
            inner_html += export_btn
        if "Import Snapshot" not in inner_html:
            inner_html += import_btn
            
        return inner_html + m.group(2)
        
    content = re.sub(tree_content_end, replace_tree, content, flags=re.DOTALL)
    
    with open(fpath, "w") as f:
        f.write(content)

print("Done")
