#!/usr/bin/env python3
"""
Upgrades the :root CSS token block in every Hawk-Eye HTML page
to the canonical production design token palette.
Also injects the hawk-theme.css link if missing.
"""

import re, os, glob

PAGES_DIR = os.path.dirname(os.path.abspath(__file__))

# ── Production :root block ──────────────────────────────────────────────────
DARK_ROOT = """        :root {
            /* ── Background ── */
            --bg:          #080d18;
            --bg-dark:     #080d18;
            --surface:     #0d1221;
            --surface2:    #111827;
            --surface-color:     #0d1221;
            --surface-highlight: #111827;

            /* ── Borders ── */
            --border:       rgba(255,255,255,0.07);
            --border-color: rgba(255,255,255,0.07);

            /* ── Text ── */
            --text:          #f0f4ff;
            --text-primary:  #f0f4ff;
            --muted:         #a0aec8;
            --text-secondary:#a0aec8;

            /* ── Primary accent (Electric Cyan) ── */
            --accent:        #00d4ff;
            --accent-purple: #7c3aed;
            --magenta:       #00d4ff;

            /* ── Status ── */
            --green:   #00e87a;
            --yellow:  #f59e0b;
            --orange:  #fb923c;
            --red:     #ff4040;
            --blue:    #3b82f6;

            /* ── Sidebar ── */
            --sidebar-bg: #040a16;
        }"""

LIGHT_ROOT = """        body.light-theme {
            --bg:          #e8edf6;
            --bg-dark:     #e8edf6;
            --surface:     #ffffff;
            --surface2:    #f5f8ff;
            --surface-color:     #ffffff;
            --surface-highlight: #f5f8ff;

            --border:       rgba(0,0,0,0.08);
            --border-color: rgba(0,0,0,0.08);

            --text:          #0a0f1e;
            --text-primary:  #0a0f1e;
            --muted:         #4a5568;
            --text-secondary:#4a5568;

            --accent:        #0095c8;
            --accent-purple: #6d28d9;
            --magenta:       #0095c8;

            --sidebar-bg: #0a1e3d;
        }"""

# --------------------------------------------------------------------------- #
ROOT_PATTERN      = re.compile(r':root\s*\{[^}]*\}', re.DOTALL)
LIGHT_PATTERN     = re.compile(r'body\.light-theme\s*\{[^}]*\}', re.DOTALL)
FONT_IMPORT       = "@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800;900&family=Fira+Code:wght@400;500;600&display=swap');"
THEME_LINK        = '<link id="hawk-theme-css-link" rel="stylesheet" href="/scripts/hawk-theme.css">'
FONTAWESOME_TAG   = '<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">'

def process_file(path):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()

    original = content

    # 1. Replace :root block
    if ROOT_PATTERN.search(content):
        content = ROOT_PATTERN.sub(DARK_ROOT, content, count=1)
    
    # 2. Replace or leave body.light-theme block
    if LIGHT_PATTERN.search(content):
        content = LIGHT_PATTERN.sub(LIGHT_ROOT, content, count=1)

    # 3. Inject hawk-theme.css link after FontAwesome (or before </head>)
    if 'hawk-theme-css-link' not in content:
        if FONTAWESOME_TAG in content:
            content = content.replace(FONTAWESOME_TAG, FONTAWESOME_TAG + '\n    ' + THEME_LINK, 1)
        else:
            content = content.replace('</head>', '    ' + THEME_LINK + '\n</head>', 1)

    # 4. Upgrade @import font to include all weights
    old_imports = [
        "@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=Fira+Code:wght@400;500&display=swap');",
        "@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');",
        "@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;600;700;800&family=Roboto+Mono:wght@400;500&display=swap');",
    ]
    for old in old_imports:
        if old in content:
            content = content.replace(old, FONT_IMPORT, 1)

    # 5. Upgrade legacy magenta/pink sidebar sub-label colors
    content = content.replace('color: #ff00ff;', 'color: var(--he-accent,#00d4ff);')
    content = content.replace("color: '#ff00ff'", "color: 'var(--he-accent,#00d4ff)'")
    content = content.replace('#ff6ef7', 'var(--he-accent,#00d4ff)')

    if content != original:
        with open(path, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f'  ✅  Updated: {os.path.basename(path)}')
    else:
        print(f'  ─   No change: {os.path.basename(path)}')

html_files = sorted(glob.glob(os.path.join(PAGES_DIR, '*.html')))
print(f'Processing {len(html_files)} HTML files...\n')
for p in html_files:
    process_file(p)

print('\nDone.')
