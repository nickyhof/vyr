#!/usr/bin/env python3
"""Generate Vyr standard library documentation as a static HTML page.

Reads all std/*.vyr files, extracts /// doc comments and fn signatures,
and generates docs/index.html.
"""

import os
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class Function:
    name: str
    params: str
    doc: str
    line: int


@dataclass
class Module:
    name: str
    path: str
    description: str
    functions: list = field(default_factory=list)


def parse_module(path: Path) -> Module:
    """Parse a .vyr file and extract doc comments + function signatures."""
    lines = path.read_text().splitlines()
    module_name = path.stem
    module_desc = ""
    functions = []

    # Extract module description from first // comment block
    for line in lines:
        if line.startswith("// "):
            desc = line[3:].strip()
            if desc.startswith("std/"):
                parts = desc.split(" — ", 1)
                if len(parts) == 2:
                    module_desc = parts[1]
            break

    doc_lines = []
    for i, line in enumerate(lines):
        stripped = line.strip()
        if stripped.startswith("/// "):
            doc_lines.append(stripped[4:])
        elif stripped.startswith("///"):
            doc_lines.append(stripped[3:])
        elif stripped.startswith("fn ") and not stripped.startswith("fn _"):
            # Extract function name and params
            match = re.match(r"fn\s+(\w+)\(([^)]*)\)", stripped)
            if match:
                name = match.group(1)
                params = match.group(2).strip()
                doc = "\n".join(doc_lines) if doc_lines else ""
                functions.append(Function(name=name, params=params, doc=doc, line=i + 1))
            doc_lines = []
        else:
            if not stripped.startswith("//") and stripped != "":
                doc_lines = []

    return Module(
        name=module_name,
        path=str(path),
        description=module_desc,
        functions=functions,
    )


def generate_html(modules: list[Module]) -> str:
    """Generate a complete HTML page from parsed modules."""
    # Sort modules in a nice order
    order = ["math", "string", "collections", "result", "functional", "io"]
    modules.sort(key=lambda m: order.index(m.name) if m.name in order else 99)

    sidebar_html = ""
    content_html = ""

    for mod in modules:
        icon = {
            "math": "∑",
            "string": "𝕊",
            "collections": "☰",
            "result": "⊕",
            "functional": "λ",
            "io": "⇄",
        }.get(mod.name, "•")

        sidebar_html += f"""
        <div class="nav-module">
          <a href="#mod-{mod.name}" class="nav-module-title" data-module="{mod.name}">
            <span class="nav-icon">{icon}</span> {mod.name}
          </a>
          <div class="nav-functions" id="nav-{mod.name}">
"""
        for fn in mod.functions:
            sidebar_html += f'            <a href="#fn-{mod.name}-{fn.name}" class="nav-fn">{fn.name}</a>\n'
        sidebar_html += "          </div>\n        </div>\n"

        content_html += f"""
      <section class="module" id="mod-{mod.name}">
        <div class="module-header">
          <span class="module-icon">{icon}</span>
          <div>
            <h2>{mod.name}</h2>
            <p class="module-desc">{mod.description}</p>
          </div>
        </div>
        <div class="fn-grid">
"""
        for fn in mod.functions:
            param_list = fn.params
            doc_html = fn.doc.replace("\n", "<br>") if fn.doc else '<span class="no-doc">No documentation.</span>'
            content_html += f"""
          <div class="fn-card" id="fn-{mod.name}-{fn.name}">
            <div class="fn-sig"><span class="kw">fn</span> <span class="fn-name">{fn.name}</span>(<span class="fn-params">{param_list}</span>)</div>
            <div class="fn-doc">{doc_html}</div>
          </div>
"""
        content_html += "        </div>\n      </section>\n"

    total_fns = sum(len(m.functions) for m in modules)

    return f"""<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Vyr Standard Library</title>
  <meta name="description" content="Complete reference for the Vyr standard library — {total_fns} functions across {len(modules)} modules.">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
    :root {{
      --bg: #0d1117;
      --bg-surface: #161b22;
      --bg-card: #1c2129;
      --bg-card-hover: #21262d;
      --border: #30363d;
      --border-accent: #3b82f6;
      --text: #e6edf3;
      --text-secondary: #8b949e;
      --text-muted: #6e7681;
      --accent: #58a6ff;
      --accent-glow: rgba(88, 166, 255, 0.15);
      --kw: #ff7b72;
      --fn-color: #d2a8ff;
      --param: #79c0ff;
      --sidebar-w: 260px;
      --radius: 10px;
    }}

    * {{ margin: 0; padding: 0; box-sizing: border-box; }}

    body {{
      font-family: 'Inter', -apple-system, sans-serif;
      background: var(--bg);
      color: var(--text);
      line-height: 1.6;
      display: flex;
      min-height: 100vh;
    }}

    /* Sidebar */
    .sidebar {{
      width: var(--sidebar-w);
      background: var(--bg-surface);
      border-right: 1px solid var(--border);
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      overflow-y: auto;
      z-index: 100;
      display: flex;
      flex-direction: column;
    }}

    .sidebar-header {{
      padding: 24px 20px 16px;
      border-bottom: 1px solid var(--border);
    }}

    .sidebar-header h1 {{
      font-size: 18px;
      font-weight: 700;
      background: linear-gradient(135deg, var(--accent), var(--fn-color));
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }}

    .sidebar-header .tag {{
      font-size: 11px;
      color: var(--text-muted);
      margin-top: 4px;
    }}

    .search-box {{
      padding: 12px 16px;
      border-bottom: 1px solid var(--border);
    }}

    .search-box input {{
      width: 100%;
      padding: 8px 12px;
      border: 1px solid var(--border);
      border-radius: 6px;
      background: var(--bg);
      color: var(--text);
      font-size: 13px;
      font-family: 'Inter', sans-serif;
      outline: none;
      transition: border-color 0.2s;
    }}

    .search-box input:focus {{
      border-color: var(--accent);
      box-shadow: 0 0 0 3px var(--accent-glow);
    }}

    .search-box input::placeholder {{
      color: var(--text-muted);
    }}

    .nav-modules {{
      padding: 12px 0;
      flex: 1;
      overflow-y: auto;
    }}

    .nav-module {{
      margin-bottom: 4px;
    }}

    .nav-module-title {{
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 20px;
      color: var(--text);
      text-decoration: none;
      font-weight: 600;
      font-size: 13px;
      transition: background-color 0.15s;
    }}

    .nav-module-title:hover {{
      background: var(--bg-card);
    }}

    .nav-icon {{
      font-size: 16px;
      width: 20px;
      text-align: center;
      opacity: 0.8;
    }}

    .nav-functions {{
      padding-left: 48px;
    }}

    .nav-fn {{
      display: block;
      padding: 3px 12px;
      color: var(--text-secondary);
      text-decoration: none;
      font-size: 12.5px;
      font-family: 'JetBrains Mono', monospace;
      border-radius: 4px;
      transition: all 0.15s;
    }}

    .nav-fn:hover {{
      color: var(--accent);
      background: var(--accent-glow);
    }}

    .nav-fn.hidden {{
      display: none;
    }}

    /* Main content */
    .main {{
      margin-left: var(--sidebar-w);
      flex: 1;
      padding: 48px;
      max-width: 960px;
    }}

    .hero {{
      margin-bottom: 48px;
    }}

    .hero h1 {{
      font-size: 36px;
      font-weight: 700;
      background: linear-gradient(135deg, var(--accent), var(--fn-color), #ff7b72);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }}

    .hero p {{
      color: var(--text-secondary);
      font-size: 16px;
      margin-top: 8px;
    }}

    .stats {{
      display: flex;
      gap: 24px;
      margin-top: 20px;
    }}

    .stat {{
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 12px 20px;
    }}

    .stat-value {{
      font-size: 24px;
      font-weight: 700;
      color: var(--accent);
    }}

    .stat-label {{
      font-size: 12px;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }}

    /* Module sections */
    .module {{
      margin-bottom: 56px;
    }}

    .module-header {{
      display: flex;
      align-items: center;
      gap: 16px;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--border);
    }}

    .module-icon {{
      font-size: 32px;
      width: 48px;
      height: 48px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: var(--bg-surface);
      border: 1px solid var(--border);
      border-radius: 12px;
    }}

    .module-header h2 {{
      font-size: 22px;
      font-weight: 700;
    }}

    .module-desc {{
      color: var(--text-secondary);
      font-size: 14px;
      margin-top: 2px;
    }}

    .fn-grid {{
      display: flex;
      flex-direction: column;
      gap: 12px;
    }}

    .fn-card {{
      background: var(--bg-card);
      border: 1px solid var(--border);
      border-radius: var(--radius);
      padding: 16px 20px;
      transition: all 0.2s ease;
    }}

    .fn-card:hover {{
      background: var(--bg-card-hover);
      border-color: var(--border-accent);
      box-shadow: 0 0 0 1px var(--accent-glow);
    }}

    .fn-card.hidden {{
      display: none;
    }}

    .fn-sig {{
      font-family: 'JetBrains Mono', monospace;
      font-size: 14px;
      margin-bottom: 8px;
    }}

    .kw {{
      color: var(--kw);
      font-weight: 500;
    }}

    .fn-name {{
      color: var(--fn-color);
      font-weight: 500;
    }}

    .fn-params {{
      color: var(--param);
    }}

    .fn-doc {{
      color: var(--text-secondary);
      font-size: 13.5px;
      line-height: 1.5;
    }}

    .no-doc {{
      color: var(--text-muted);
      font-style: italic;
    }}

    /* Mobile */
    @media (max-width: 768px) {{
      .sidebar {{
        transform: translateX(-100%);
        transition: transform 0.3s;
      }}

      .sidebar.open {{
        transform: translateX(0);
      }}

      .main {{
        margin-left: 0;
        padding: 24px 16px;
      }}

      .hero h1 {{ font-size: 28px; }}
      .stats {{ flex-wrap: wrap; gap: 12px; }}
    }}

    /* Scrollbar */
    ::-webkit-scrollbar {{ width: 6px; }}
    ::-webkit-scrollbar-track {{ background: transparent; }}
    ::-webkit-scrollbar-thumb {{ background: var(--border); border-radius: 3px; }}
    ::-webkit-scrollbar-thumb:hover {{ background: var(--text-muted); }}

    /* Smooth scroll */
    html {{ scroll-behavior: smooth; }}

    /* Scroll offset for fixed sidebar */
    :target {{ scroll-margin-top: 24px; }}
  </style>
</head>
<body>
  <nav class="sidebar">
    <div class="sidebar-header">
      <h1>vyr stdlib</h1>
      <div class="tag">{total_fns} functions · {len(modules)} modules</div>
    </div>
    <div class="search-box">
      <input type="text" id="search" placeholder="Search functions…" autocomplete="off">
    </div>
    <div class="nav-modules">
{sidebar_html}
    </div>
  </nav>

  <main class="main">
    <div class="hero">
      <h1>Vyr Standard Library</h1>
      <p>Complete reference for every function in the standard library.</p>
      <div class="stats">
        <div class="stat">
          <div class="stat-value">{len(modules)}</div>
          <div class="stat-label">Modules</div>
        </div>
        <div class="stat">
          <div class="stat-value">{total_fns}</div>
          <div class="stat-label">Functions</div>
        </div>
      </div>
    </div>

{content_html}
  </main>

  <script>
    const search = document.getElementById('search');
    const cards = document.querySelectorAll('.fn-card');
    const navFns = document.querySelectorAll('.nav-fn');

    search.addEventListener('input', () => {{
      const q = search.value.toLowerCase().trim();
      cards.forEach(card => {{
        const name = card.querySelector('.fn-name').textContent.toLowerCase();
        const doc = card.querySelector('.fn-doc').textContent.toLowerCase();
        const match = !q || name.includes(q) || doc.includes(q);
        card.classList.toggle('hidden', !match);
      }});
      navFns.forEach(link => {{
        const name = link.textContent.toLowerCase();
        link.classList.toggle('hidden', q && !name.includes(q));
      }});
    }});

    // Keyboard shortcut: / to focus search
    document.addEventListener('keydown', e => {{
      if (e.key === '/' && document.activeElement !== search) {{
        e.preventDefault();
        search.focus();
      }}
      if (e.key === 'Escape') {{
        search.blur();
        search.value = '';
        search.dispatchEvent(new Event('input'));
      }}
    }});
  </script>
</body>
</html>"""


def main():
    root = Path(__file__).parent.parent
    std_dir = root / "std"
    docs_dir = root / "docs"

    if not std_dir.exists():
        print(f"Error: {std_dir} not found", file=sys.stderr)
        sys.exit(1)

    modules = []
    for vyr_file in sorted(std_dir.glob("*.vyr")):
        mod = parse_module(vyr_file)
        if mod.functions:
            modules.append(mod)
            print(f"  {mod.name}: {len(mod.functions)} functions")

    docs_dir.mkdir(exist_ok=True)
    html = generate_html(modules)
    out_path = docs_dir / "index.html"
    out_path.write_text(html)
    print(f"\nGenerated {out_path} ({len(modules)} modules, {sum(len(m.functions) for m in modules)} functions)")


if __name__ == "__main__":
    main()
