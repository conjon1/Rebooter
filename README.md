# Rebooter — AI Reference README

> This document is written for AI assistants (and humans) to quickly understand the Rebooter codebase. When asked to modify, extend, or debug this project, read this first.

---

## What Is This?

**Rebooter** is a bespoke Static Site Generator (SSG) written in Go. It produces a single-page blog application (`public_html/index.html`) from raw HTML post and author files. There is no database, no CMS, and no runtime server — just a Go build script and a Templ template.

Live site: `Rebooter.blog`
Authors: Connal McInnis, Keshav Italia, Tanveer Salim

---

## Tech Stack

| Layer        | Technology                                      |
|--------------|-------------------------------------------------|
| Language     | Go 1.24                                         |
| Templating   | [Templ](https://templ.guide/) v0.3.960          |
| HTML Parsing | `golang.org/x/net/html`                         |
| Frontend CSS | Tailwind CSS (CDN)                              |
| Frontend JS  | Vanilla JS (zero frameworks, SPA router)        |
| Fonts        | Fira Code (mono), Inter (sans) via Google Fonts |

---

## Repository Layout

```
Rebooter/
├── build.go                  # Main build script — entry point
├── components.templ          # Templ source: HTML templates & components
├── components_templ.go       # Auto-generated Go from components.templ (do NOT hand-edit)
├── go.mod
├── go.sum
├── DO_NOT_UPLOAD_HERE        # Sentinel file — keep secrets/large assets out of root
└── public_html/              # OUTPUT directory — served by web server
    ├── index.html            # Generated SPA (output of build)
    ├── posts.json            # Generated post metadata (output of build)
    ├── authors.json          # Generated author metadata (output of build)
    ├── Connal_McInnis_-_Resume.pdf
    ├── posts/                # Raw HTML blog post files (SOURCE — hand-written)
    │   ├── K3S.html
    │   ├── TmobileSpoof.html
    │   ├── DockerIsNotASandbox.html
    │   └── ...
    └── authors/              # Raw HTML author profile files (SOURCE — hand-written)
        ├── connalmcinnis.html
        ├── keshavitalia.html
        └── tanveersalim.html
```

**Golden rule:** `public_html/posts/` and `public_html/authors/` are the *source of truth*. Everything else in `public_html/` is generated or static.

---

## How the Build Works (`build.go`)

Running `go run .` from the project root executes this pipeline:

```
1. Scan public_html/posts/*.html
      → parse <title> and <meta name="date|excerpt|author"> from each file
      → produce []Post{ID, Title, Date, Excerpt, URL, Author, AuthorID}

2. Scan public_html/authors/*.html
      → parse <meta name="name|linkedin|github"> from each file
      → produce []AuthorData{ID, Name, Linkedin, Github}

3. Sort posts newest-first by Date ("January 2, 2006" format)

4. Cross-link: set Post.AuthorID by matching Post.Author to AuthorData.Name

5. Write public_html/posts.json   (pretty-printed)
   Write public_html/authors.json (pretty-printed)

6. Render components.templ:Index(posts, authors) → public_html/index.html
```

### Key Structs

```go
type Post struct {
    ID       string  // filename without .html  (e.g. "K3S")
    Title    string  // from <title> tag
    Date     string  // meta name="date"         format: "January 2, 2006"
    Excerpt  string  // meta name="excerpt"
    URL      string  // "posts/<filename>.html"
    Author   string  // meta name="author"       (display name)
    AuthorID string  // derived from authors dir (filename without .html)
}

type AuthorData struct {
    ID       string  // filename without .html  (e.g. "connalmcinnis")
    Name     string  // meta name="name"
    Linkedin string  // meta name="linkedin"
    Github   string  // meta name="github"
}
```

---

## How to Add a New Blog Post

Create `public_html/posts/<slug>.html`. Required minimum:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Your Post Title Here</title>
    <meta name="date"    content="March 14, 2026">
    <meta name="excerpt" content="One sentence summary shown on the index page.">
    <meta name="author"  content="Connal McInnis">
    <!-- optional: <meta name="tags" content="linux, go"> -->
</head>
<body>
    <!-- Post content goes here. Use plain HTML. -->
    <!-- Styling is inherited from the parent SPA's .prose CSS class. -->
</body>
</html>
```

**Rules:**
- `<title>` is required. Missing title → post is skipped with a warning.
- `meta name="date"` is required. Format **must** be `"January 2, 2006"` (Go's reference time).
- `meta name="author"` must exactly match an `AuthorData.Name` value for cross-linking to work. If missing, author defaults to `"UNKNOWN"`.
- Slug becomes the post ID. URL will be `#post/<slug>`.
- Then run `go run .` to rebuild.

---

## How to Add a New Author

Create `public_html/authors/<authorid>.html`. The filename (without `.html`) becomes the author's ID.

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta name="name"     content="Jane Doe">
    <meta name="linkedin" content="https://linkedin.com/in/janedoe">
    <meta name="github"   content="https://github.com/janedoe">
</head>
<body>
    <!-- Optional: author bio / profile content rendered at #author/<authorid> -->
</body>
</html>
```

**Rules:**
- `meta name="name"` is required. Missing name → author is skipped.
- The `name` value must match exactly what post files use in `meta name="author"`.

---

## Frontend SPA (`components.templ`)

The generated `index.html` is a single-page app driven by `window.location.hash`.

### Pages / Sections

| Hash pattern      | Shows              | Description                        |
|-------------------|--------------------|------------------------------------|
| `#home`           | Home section       | Hero + 3 most recent posts         |
| `#blog`           | Blog section       | All posts + live search input      |
| `#authors`        | Authors section    | Author list with per-author posts  |
| `#post/<id>`      | Content view       | Fetches `posts/<id>.html` via AJAX |
| `#author/<id>`    | Content view       | Fetches `authors/<id>.html` via AJAX |

### JavaScript class: `BlogApp`

```
BlogApp
 ├── init()             → fetchAuthors(), initRouter(), initMobileMenu()
 ├── initRouter()       → listens to hashchange, calls handleRoute()
 ├── handleRoute()      → dispatches to showPage / loadPost / loadAuthorPage
 ├── showPage(id)       → hides all .page sections, shows the target
 ├── loadPost(id)       → fetch posts/<id>.html, inject author byline, render into #dynamic-content
 ├── loadAuthorPage(id) → fetch authors/<id>.html, render body into #dynamic-content
 ├── initSearch()       → live filters #all-posts articles by title + excerpt
 ├── initTypewriter()   → cycles role strings in the hero heading
 ├── initObservers()    → IntersectionObserver for .fade-in animations
 └── initMobileMenu()   → hamburger toggle
```

### Templ Components

| Component            | Description                                               |
|----------------------|-----------------------------------------------------------|
| `Index(posts, authors)` | Root layout — head, nav, all sections, footer, JS    |
| `PostItem(post)`     | Reusable article card used in Home, Blog, and Authors sections |

### Color Palette (CSS variables)

```css
--bg:        #1e2127   /* page background */
--bg-alt:    #282c34   /* card / code block background */
--fg:        #abb2bf   /* body text */
--fg-alt:    #5c6370   /* muted text, borders */
--primary:   #61afef   /* headings, links, highlights (blue) */
--secondary: #c678dd   /* h2, author links, read-more (purple) */
--accent:    #98c379   /* h3, post titles (green) */
--warning:   #d19a66   /* strong, emphasized text (orange) */
--error:     #e06c75   /* code inline, error states (red) */
```
Theme is One Dark Pro inspired.

---

## Templ Workflow

> **Important:** `components_templ.go` is auto-generated. Never edit it by hand.

```bash
# Install templ CLI (once)
go install github.com/a-h/templ/cmd/templ@latest

# After editing components.templ, regenerate:
templ generate

# Then build the site:
go run .
```

If you only change `build.go` (no template changes), you can skip `templ generate`.

---

## Running Locally

```bash
# From project root
go run .

# Serve the output (any static file server works):
cd public_html && python3 -m http.server 8080
# Open http://localhost:8080
```

---

## Post Styling Reference

Post HTML files use plain HTML. When loaded into the SPA, they are wrapped in a `div.prose` container which provides these styles:

| Element       | Style                                          |
|---------------|------------------------------------------------|
| `h1`          | 2rem, color: `--primary` (blue)               |
| `h2`          | 1.5rem, color: `--secondary` (purple)         |
| `h3`          | 1.25rem, color: `--accent` (green)            |
| `p`           | line-height 1.7, margin-bottom 1rem           |
| `code`        | Fira Code, bg `#2c313a`, color `--error` (red)|
| `pre`         | bg `--bg-alt`, padding, rounded, scrollable   |
| `blockquote`  | left border `--fg-alt`, italic, muted color   |
| `table`       | full-width, rounded, `--bg-alt` bg            |
| `a`           | color `--secondary`, underline                |

Posts can include their own `<style>` block for additional standalone styling (used when reading the HTML file directly), but this is ignored by the SPA since only `<body>` content is injected.

---

## Common Mistakes & Gotchas

1. **Date format** — Must be `"January 2, 2006"` (Go's reference time). Wrong format → post sorts to the bottom or errors silently.
2. **Author name mismatch** — `meta name="author"` in a post must be the exact string stored in the author file's `meta name="name"`. Case-sensitive.
3. **Editing `components_templ.go`** — Don't. Run `templ generate` instead after editing `components.templ`.
4. **`public_html/posts/` vs root** — Post HTML files live in `public_html/posts/`, **not** the project root.
5. **Rebuild after changes** — Adding/editing posts or authors requires running `go run .` to regenerate `index.html` and the JSON files.
6. **`DO_NOT_UPLOAD_HERE`** — This file in the project root is a warning marker; don't commit sensitive files or large binaries to the project root.

---

## Current Authors

| ID              | Display Name    | GitHub                            |
|-----------------|-----------------|-----------------------------------|
| `connalmcinnis` | Connal McInnis  | github.com/conjon1                |
| `keshavitalia`  | Keshav Italia   | github.com/Keshav25               |
| `tanveersalim`  | Tanveer Salim   | github.com/fosres                 |

---

## Current Posts (as of last build)

| Slug                  | Title                                                         | Author          | Date              |
|-----------------------|---------------------------------------------------------------|-----------------|-------------------|
| `TmobileSpoof`        | T-mobile TTL spoofing                                         | Connal McInnis  | January 6, 2026   |
| `CertCIndustries`     | Industries in Demand for CERT C Compliant Security Coders     | Tanveer Salim   | November 22, 2025 |
| `K3S`                 | My Homelab Journey: From Mini PCs to Kubernetes with K3s      | Connal McInnis  | October 20, 2025  |
| `DockerIsNotASandbox` | (see file)                                                    | (see file)      | (see file)        |
