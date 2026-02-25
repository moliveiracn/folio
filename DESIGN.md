# folio — Design Document

> A self-hosted, keyboard-driven document reader for PDF and EPUB.  
> Minimal interface. Persistent marks & highlights. Sioyek spirit in the browser.

**Status:** Final MVP Specification (ready for implementation)  
**Version:** 0.1  
**Inspired by:** [sioyek](https://github.com/ahrm/sioyek)  
**Date:** February 24, 2026

---

## Table of Contents

1. [Vision](#1-vision)
2. [Design Philosophy](#2-design-philosophy)
3. [Architecture Overview](#3-architecture-overview)
4. [Project Structure](#4-project-structure)
5. [Data Model](#5-data-model)
6. [CLI Commands](#6-cli-commands)
7. [API Specification](#7-api-specification)
8. [Frontend Architecture](#8-frontend-architecture)
9. [Keyboard Reference](#9-keyboard-reference)
10. [Feature Roadmap](#10-feature-roadmap)
11. [Deployment](#11-deployment)

---

## 1. Vision

Folio inverts the usual self-hosted reading tools: the reader is the product, and the library is just a doorway into it.

It brings Sioyek's keyboard-first, research-oriented experience to the browser — self-hosted, single-user, accessible from any device on your network. Open once, read forever, with your marks and highlights always there.

**What Folio is:**
- A focused, near-chromeless reading environment for PDFs and EPUBs
- 95% keyboard-driven — Vim + Sioyek style interaction model
- Single-binary, zero external services, trivial backup
- A permanent home for your personal annotations

**What Folio is not:**
- A Calibre-style library manager
- A social or cloud platform
- Bloated or Electron-heavy

---

## 2. Design Philosophy

**The document owns the screen.** At rest, only the document is visible. No sidebar, no toolbar, no breadcrumbs. UI chrome appears only when invoked and disappears when dismissed.

**Keyboard over mouse.** Every action has a shortcut. The mouse is supported but never required. Overlay navigation (search results, bookmark lists, ToC) is always keyboard-navigable with `j`/`k` and `Enter`.

**Overlays, not pages.** Switching between library, settings, and reader does not navigate to a new page — it opens and closes overlays within the same viewport. The document is always underneath.

**Persistence by default.** Reading position is saved continuously. Returning to a document always resumes exactly where you left off, across sessions and devices.

**One file, one truth.** All state lives in a single SQLite file (`folio.db`). Backup is `cp folio.db folio.db.bak`. No external databases, no Redis, no message queues.

**Minimal visual language.** Dark-first, two themes (dark / light) plus a sepia reading modifier. System monospace for all UI elements — overlays, status bar, command palette. No gradients. No drop shadows. Transitions capped at 120ms.

**Single-user, mandatory password.** Modeled after code-server: `folio init` generates a random password once, prints it to stdout, and hashes it with bcrypt. No anonymous access — Folio is designed to be reachable from outside your LAN via Cloudflare Tunnel.

---

## 3. Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                             │
│                                                             │
│  ┌──────────────────┐  ┌──────────────────────────────────┐ │
│  │  Library Overlay │  │        Reader Viewport           │ │
│  │  (fuzzy list)    │  │                                  │ │
│  └──────────────────┘  │  ┌────────────┐ ┌─────────────┐  │ │
│                        │  │  PDF.js    │ │ EPUB        │  │ │
│  ┌──────────────────┐  │  │  renderer  │ │ renderer    │  │ │
│  │  Command Palette │  │  └────────────┘ └─────────────┘  │ │
│  └──────────────────┘  │                                  │ │
│                        │  Keyboard Dispatcher             │ │
│  ┌──────────────────┐  │  Annotation Layer (SVG)          │ │
│  │  Annotation      │  │  Status Bar                      │ │
│  │  Overlay         │  └──────────────────────────────────┘ │
│  └──────────────────┘                                       │
└───────────────────────────────┬─────────────────────────────┘
                                │ HTTP
┌───────────────────────────────▼─────────────────────────────┐
│                         Go Server                           │
│                                                             │
│  Chi Router                                                 │
│  ├── /login                  Password form (session cookie) │
│  ├── /                       Library shell (HTML)           │
│  ├── /viewer/{doc_id}        Viewer shell (HTML)            │
│  ├── /api/documents          Document CRUD + upload         │
│  ├── /api/highlights         Highlight CRUD                 │
│  ├── /api/bookmarks          Bookmark CRUD                  │
│  ├── /api/marks              Mark CRUD                      │
│  ├── /api/positions          Reading position persistence   │
│  ├── /api/search             In-document + library search   │
│  └── /data/books/{id}        Raw file serving (ranges)      │
│                                                             │
│  SQLite (modernc.org/sqlite — pure Go, no CGo)              │
│  File storage: {data_dir}/books/                            │
└─────────────────────────────────────────────────────────────┘
```

### Stack decisions

**Backend: Go 1.23+ with chi router**  
Compiles to a single static binary. Low idle memory (~15MB). No framework beyond a router. Pure-Go sqlite driver (`modernc.org/sqlite`) means clean cross-compilation with no CGo.

**Frontend: Vanilla JS + ES Modules**  
No build step, no bundler, no framework. Modules loaded natively by the browser. The DOM surface is small and mostly static — complexity lives in event handling and state, not in rendering trees. External dependencies limited to `pdf.js` (PDF rendering) and `fuse.js` (fuzzy search in overlays).

**PDF rendering: PDF.js v4+**  
Runs entirely in the browser. Handles the text layer needed for selection → highlight. Canvas-based output allows an SVG annotation layer on top.

**EPUB rendering: Server-side extraction + custom renderer**  
EPUBs are zip archives of HTML chapters. The Go server extracts, sanitizes, and serves each chapter as clean HTML. The frontend injects chapter content into a controlled reading container with typed styles. This keeps the client simple and gives us full control over typography, theme injection, and annotation anchoring without delegating to a third-party EPUB runtime.

**Auth: bcrypt session cookie**  
`folio init` generates a cryptographically random password, prints it once, stores a bcrypt hash in `config.yaml`. Login issues a signed session cookie. No JWT, no OAuth — one user, one secret.

---

## 4. Project Structure

```
folio/
├── cmd/
│   └── folio/
│       └── main.go                 # CLI entry point: init, serve, passwd
│
├── internal/
│   ├── cli/
│   │   ├── init.go                 # folio init: create dirs, DB, config, password
│   │   ├── serve.go                # folio serve: start HTTP server
│   │   └── passwd.go               # folio passwd: change password
│   │
│   ├── handler/
│   │   ├── auth.go                 # /login GET+POST, session middleware
│   │   ├── library.go              # / (library shell + document list)
│   │   ├── viewer.go               # /viewer/{doc_id} (viewer shell)
│   │   ├── documents.go            # /api/documents CRUD + upload
│   │   ├── highlights.go           # /api/highlights CRUD
│   │   ├── bookmarks.go            # /api/bookmarks CRUD
│   │   ├── marks.go                # /api/marks CRUD
│   │   ├── positions.go            # /api/positions upsert + get
│   │   └── search.go               # /api/search
│   │
│   ├── middleware/
│   │   └── auth.go                 # session cookie validation
│   │
│   ├── model/
│   │   └── models.go               # shared structs: Document, Highlight, Mark, etc.
│   │
│   ├── repo/
│   │   ├── repo.go                 # DB interface
│   │   ├── sqlite.go               # modernc sqlite implementation
│   │   └── migrations/
│   │       ├── 001_init.sql
│   │       └── 002_indexes.sql
│   │
│   ├── service/
│   │   ├── documents.go            # upload processing, metadata extraction
│   │   ├── epub.go                 # EPUB extraction, chapter serving, sanitization
│   │   └── search.go               # in-doc + cross-library search logic
│   │
│   └── config/
│       └── config.go               # config.yaml struct + loader
│
├── web/
│   ├── static/
│   │   ├── pdf.js/                 # vendored PDF.js v4
│   │   ├── fuse.min.js             # fuzzy search
│   │   └── app.css                 # base styles + theme variables
│   │
│   ├── src/                        # ES modules (no build step)
│   │   ├── main.js                 # app bootstrap, route detection
│   │   ├── reader/
│   │   │   ├── pdf.js              # PDF.js integration + page management
│   │   │   ├── epub.js             # EPUB chapter fetch + DOM injection
│   │   │   └── annotations.js      # SVG overlay, highlight draw/apply
│   │   ├── ui/
│   │   │   ├── overlay.js          # base overlay: mount, focus trap, Esc
│   │   │   ├── library.js          # library overlay (fuzzy list)
│   │   │   ├── bookmarks.js        # bookmark overlay
│   │   │   ├── toc.js              # table of contents overlay
│   │   │   ├── search.js           # search overlay
│   │   │   ├── commandpalette.js   # : command palette
│   │   │   └── statusbar.js        # bottom status bar
│   │   ├── keyboard/
│   │   │   ├── dispatcher.js       # central keydown handler (FSM)
│   │   │   ├── keymap.js           # default keybindings
│   │   │   └── sequences.js        # multi-key: gg, mA, hr...
│   │   └── api/
│   │       └── client.js           # typed fetch wrappers for all endpoints
│   │
│   └── templates/
│       ├── base.html               # layout, <head>, theme class
│       ├── login.html
│       ├── library.html
│       └── viewer.html
│
├── data/                           # .gitignored — Docker volume
│   ├── books/
│   ├── folio.db
│   └── config.yaml
│
├── docs/
│   ├── schema.sql                  # canonical schema (mirrors migrations)
│   └── KEYBINDINGS.md
│
├── config.example.yaml
├── Dockerfile
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## 5. Data Model

All tables live in `folio.db`. Migrations are embedded SQL files executed in order at server startup. The canonical schema is also kept at `docs/schema.sql` for reference.

### `docs`

```sql
CREATE TABLE docs (
    id          TEXT PRIMARY KEY,   -- SHA256 of file content (auto-deduplicates)
    filename    TEXT NOT NULL,       -- original upload filename
    title       TEXT,                -- extracted from metadata or user-set
    author      TEXT,
    format      TEXT NOT NULL CHECK (format IN ('pdf', 'epub')),
    page_count  INTEGER,             -- null for epub until parsed; chapter count for epub
    file_size   INTEGER NOT NULL,
    uploaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_opened DATETIME
);
```

Using SHA256 as the primary key means uploading the same file twice is a no-op at the DB level. The file is also stored at `books/{sha256}.{ext}` — content-addressed, no collisions.

### `positions`

```sql
CREATE TABLE positions (
    doc_id      TEXT PRIMARY KEY REFERENCES docs(id) ON DELETE CASCADE,
    page        INTEGER NOT NULL DEFAULT 1,
    scroll_y    REAL NOT NULL DEFAULT 0.0,    -- fractional scroll within page (0.0–1.0)
    chapter     TEXT,                         -- epub chapter href; null for pdf
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Kept separate from `docs` so the position upsert path (`PUT /api/positions/:id`) never touches the documents table. Clean separation of concern, no accidental overwrites of metadata.

### `marks`

```sql
-- Vim-style single-character marks.
-- Lowercase (a-z): scoped to one document.
-- Uppercase (A-Z): global, stored in global_marks.
CREATE TABLE marks (
    doc_id      TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    key         TEXT NOT NULL CHECK (length(key) = 1),   -- 'a'–'z'
    page        INTEGER,
    scroll_y    REAL NOT NULL DEFAULT 0.0,
    chapter     TEXT,
    PRIMARY KEY (doc_id, key)
);

CREATE TABLE global_marks (
    key         TEXT PRIMARY KEY CHECK (length(key) = 1), -- 'A'–'Z'
    doc_id      TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    page        INTEGER,
    scroll_y    REAL NOT NULL DEFAULT 0.0,
    chapter     TEXT
);
```

### `bookmarks`

```sql
CREATE TABLE bookmarks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_id      TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,
    label       TEXT NOT NULL,
    page        INTEGER,
    scroll_y    REAL NOT NULL DEFAULT 0.0,
    chapter     TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bookmarks_doc ON bookmarks(doc_id);
```

### `highlights`

```sql
CREATE TABLE highlights (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_id          TEXT NOT NULL REFERENCES docs(id) ON DELETE CASCADE,

    -- PDF: page number + character range within PDF.js text layer
    page            INTEGER,
    start_offset    INTEGER,
    end_offset      INTEGER,

    -- EPUB: chapter href + CSS selector path + char offset within element
    chapter         TEXT,
    start_xpath     TEXT,
    end_xpath       TEXT,

    text            TEXT NOT NULL,   -- the highlighted string (enables text search)
    color           TEXT NOT NULL DEFAULT 'yellow'
                        CHECK (color IN ('yellow', 'red', 'green', 'blue')),
    note            TEXT,            -- optional free-text annotation
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_highlights_doc ON highlights(doc_id);
```

### Open question: EPUB highlight anchoring

EPUB content is reflowed HTML — character offsets drift if viewport width, font size, or chapter content change. Two candidate strategies:

- **xpath + char offset** (chosen for MVP): `(CSS selector path to element, start offset, end offset within element text)`. Simple to implement, but fragile if the chapter DOM structure changes.
- **EPUB CFI** (Canonical Fragment Identifier): spec-compliant, viewport-independent. Complex to implement; deferred to post-MVP.

The `start_xpath` / `end_xpath` columns are named generically so CFI strings can be stored there later without a schema migration.

---

## 6. CLI Commands

`folio` is the single binary. Subcommands handle the full lifecycle.

```bash
# First run — creates data dir, folio.db, config.yaml, generates password
folio init --data ./data --port 8080

# Or set a password explicitly (useful for scripted deploys)
folio init --data ./data --password mysecret

# Start the server (reads config from --data/config.yaml)
folio serve --data ./data

# Change password interactively
folio passwd --data ./data
```

**`folio init` behaviour:**
1. Creates `{data}/books/` directory
2. Creates `folio.db` and runs all migrations
3. Generates a 20-character random password (if `--password` not provided)
4. Hashes it with bcrypt (cost 12) and writes to `config.yaml`
5. Prints the plaintext password **once** to stdout — it is never shown again

```
folio initialized.

  Data dir : ./data
  Port     : 8080
  Password : Kx9!mRqTv2LpNzWa8jYc

Keep this password safe. It will not be shown again.
Run: folio serve --data ./data
```

**`config.yaml` structure:**
```yaml
port: 8080
data_dir: ./data
password_hash: "$2a$12$..."   # bcrypt hash
log_level: info
max_upload_mb: 200
```

---

## 7. API Specification

**Base path:** `/api`  
**Auth:** All `/api/*` routes require a valid session cookie (set at `/login`).  
**Content-Type:** `application/json` for all requests and responses.  
**Errors:** `{ "error": "human-readable message" }`

---

### Auth

#### `GET /login`
Returns the login page (HTML). Redirects to `/` if already authenticated.

#### `POST /login`
```json
{ "password": "Kx9!mRqTv2LpNzWa8jYc" }
```
Sets a signed session cookie on success. Returns `401` on wrong password.  
No rate limiting in MVP — add a simple in-memory counter in Phase 2.

#### `POST /logout`
Clears the session cookie.

---

### Documents

#### `GET /api/documents`
List all documents ordered by `last_opened DESC, uploaded_at DESC`.

```json
[
  {
    "id": "a3f8c2d1...",
    "filename": "thinking-fast-and-slow.pdf",
    "title": "Thinking, Fast and Slow",
    "author": "Daniel Kahneman",
    "format": "pdf",
    "page_count": 499,
    "file_size": 4201832,
    "uploaded_at": "2026-02-10T14:00:00Z",
    "last_opened": "2026-02-24T09:15:00Z",
    "position": {
      "page": 142,
      "scroll_y": 0.33,
      "chapter": null
    }
  }
]
```

The `position` field is a left join — `null` if the document has never been opened.

#### `POST /api/documents`
Upload a new document. `multipart/form-data`, field name `file`.

- Computes SHA256 of content → becomes the document ID
- Returns `200` with existing document if hash already exists (idempotent dedup)
- Returns `201` with new document on first upload
- Extracts title/author from PDF metadata or EPUB `content.opf`
- Stores file at `{data_dir}/books/{sha256}.{pdf|epub}`

#### `GET /api/documents/:id`
Single document metadata + position.

#### `PATCH /api/documents/:id`
Override extracted metadata:
```json
{ "title": "Custom Title", "author": "Custom Author" }
```

#### `DELETE /api/documents/:id`
Deletes document row, all annotations, and the file on disk.

---

### EPUB Chapters

#### `GET /api/documents/:id/chapters`
List chapters in reading order.

```json
[
  { "href": "ch01.xhtml", "title": "Chapter 1", "order": 1 },
  { "href": "ch02.xhtml", "title": "Chapter 2", "order": 2 }
]
```

#### `GET /api/documents/:id/chapters/:href`
Returns a single chapter as sanitized HTML — stripped of `<script>`, external resources rewritten or removed, CSS scoped to prevent bleed into the host page. The frontend injects this directly into the reading container.

---

### Raw File Serving

#### `GET /data/books/:id`
Serves the raw file. Supports `Range` requests — required by PDF.js for progressive loading of large PDFs.

---

### Positions

#### `GET /api/positions/:doc_id`
```json
{
  "doc_id": "a3f8c2d1...",
  "page": 142,
  "scroll_y": 0.33,
  "chapter": null,
  "updated_at": "2026-02-24T09:15:00Z"
}
```
Returns `404` if no position has been saved yet (document never opened).

#### `PUT /api/positions/:doc_id`
Upsert. Called by the frontend every 5 seconds while reading.
```json
{ "page": 143, "scroll_y": 0.0, "chapter": null }
```

---

### Marks

#### `GET /api/marks/:doc_id`
All local marks for a document.
```json
[{ "key": "m", "page": 59, "scroll_y": 0.5, "chapter": null }]
```

#### `PUT /api/marks/:doc_id/:key`
Set or update a local mark (key must be `a–z`).
```json
{ "page": 59, "scroll_y": 0.5, "chapter": null }
```

#### `DELETE /api/marks/:doc_id/:key`

#### `GET /api/marks/global`
All global marks.
```json
[{ "key": "A", "doc_id": "a3f8c2d1...", "page": 12, "scroll_y": 0.1, "chapter": null }]
```

#### `PUT /api/marks/global/:key`
Key must be `A–Z`.
```json
{ "doc_id": "a3f8c2d1...", "page": 12, "scroll_y": 0.1, "chapter": null }
```

#### `DELETE /api/marks/global/:key`

---

### Bookmarks

#### `GET /api/bookmarks?doc_id=:id`
If `doc_id` provided: bookmarks for that document.  
If omitted: all bookmarks across the library.

#### `POST /api/bookmarks`
```json
{
  "doc_id": "a3f8c2d1...",
  "label": "Good definition of cognitive ease",
  "page": 59,
  "scroll_y": 0.71,
  "chapter": null
}
```

#### `PATCH /api/bookmarks/:id`
Update label only.

#### `DELETE /api/bookmarks/:id`

---

### Highlights

#### `GET /api/highlights?doc_id=:id`
If `doc_id` provided: highlights for that document.  
If omitted: all highlights across library (for the highlights overlay in cross-doc mode).

#### `POST /api/highlights`
PDF:
```json
{
  "doc_id": "a3f8c2d1...",
  "page": 59,
  "start_offset": 1402,
  "end_offset": 1567,
  "text": "Cognitive ease is both a cause and a consequence of a pleasant feeling.",
  "color": "yellow",
  "note": null
}
```

EPUB:
```json
{
  "doc_id": "b9e1a4f2...",
  "chapter": "ch03.xhtml",
  "start_xpath": "p:nth-child(4)",
  "end_xpath": "p:nth-child(4)",
  "text": "The experiencing self does not have a voice.",
  "color": "green",
  "note": "Kahneman's key distinction"
}
```

#### `PATCH /api/highlights/:id`
Update `color` or `note`.

#### `DELETE /api/highlights/:id`

---

### Search

#### `GET /api/search?q=:query&doc_id=:id`
`doc_id` is optional. Searches highlights and bookmarks text via SQLite `LIKE`.  
Full-text document content search is Phase 3 (FTS5).

```json
{
  "results": [
    {
      "type": "highlight",
      "id": 42,
      "doc_id": "a3f8c2d1...",
      "doc_title": "Thinking, Fast and Slow",
      "page": 59,
      "chapter": null,
      "text": "Cognitive ease is both a cause..."
    },
    {
      "type": "bookmark",
      "id": 7,
      "doc_id": "a3f8c2d1...",
      "doc_title": "Thinking, Fast and Slow",
      "page": 59,
      "label": "Good definition of cognitive ease"
    }
  ]
}
```

---

## 8. Frontend Architecture

### No framework

Vanilla JS with ES modules. No build step, no bundler, no transpiler. All files are loaded directly by the browser via `<script type="module">`. The two permitted external libraries are `pdf.js` (vendored) and `fuse.js` (vendored). Everything else is hand-rolled.

### AppState

A single module-level object, mutated only through explicit setters:

```js
// src/state.js
export const AppState = {
  doc: null,          // { id, format, pageCount, title }
  page: 1,
  chapter: null,      // epub chapter href
  zoom: 1.0,
  theme: 'dark',      // 'dark' | 'light' | 'sepia'
  overlays: [],       // stack: last item is the visible overlay
};

export function setState(patch) {
  Object.assign(AppState, patch);
}
```

No reactive proxies. Components that need to react to state changes are explicitly notified by the code that calls `setState`. Keeps the data flow visible and debuggable.

### Keyboard Dispatcher

Central `keydown` listener on `document`. Implemented as a finite state machine:

```
State: {
  sequence : string[]    // accumulated keys e.g. ['g', 'g'] or ['m', 'A']
  numPrefix: string      // accumulated digit prefix e.g. '142'
  mode     : 'normal' | 'input'
}

On keydown:
  1. If mode === 'input' → ignore (active <input> handles it)
  2. If digit and no active sequence → append to numPrefix
  3. Append key to sequence
  4. Check keymap for exact match → execute action(numPrefix), reset
  5. Check keymap for prefix match → wait (timeout 1 000 ms)
  6. Timeout or no match → reset sequence and numPrefix
```

Sequences registered in `keymap.js` as a flat map: `{ 'gg': actions.firstPage, 'mA': actions.setGlobalMark, ... }`. Actions are plain async functions that receive the numeric prefix as argument.

### Overlay System

A single `Overlay` class handles: mounting into a fixed container, focus trap, `Escape` to close. All overlays (library, bookmarks, ToC, command palette) are instances of this class with a custom `render()` method.

The overlay container sits above the reader at `z-index: 100`. At most one overlay is visible at a time; opening a second one closes the first.

### PDF Annotation Layer

A transparent `<svg>` element is sized and positioned to exactly match the PDF.js `<canvas>`. Highlights are drawn as `<rect>` elements with `fill-opacity: 0.3`. When a page is rendered, the dispatcher fetches highlights for that page and maps character offsets to page-relative coordinates using PDF.js's `getTextContent()` and `Viewport` APIs.

### EPUB Chapter Renderer

```
user presses ] or J
  → epub.js fetches /api/documents/:id/chapters/:href
  → response is sanitized HTML
  → injected into #chapter-container innerHTML
  → highlights for this chapter re-applied as <mark> elements
  → scroll to top (or to saved scroll_y if resuming)
  → position saved via PUT /api/positions/:id
```

Highlights in EPUB are `<mark data-highlight-id="42">` wrappers applied after injection via a DOM walk against the stored xpath + offset.

### Status Bar

One fixed line at the bottom. Auto-hides after 3 seconds of inactivity, reappears on any keypress or mouse move.

```
Page 142 / 499   |   Marks: 3   |   Highlights: 7   |   thinking-fast-and-slow.pdf
```

---

## 9. Keyboard Reference

Full reference lives in `docs/KEYBINDINGS.md`. This section covers the MVP set.

### Notation
- `[N]` — optional numeric prefix (e.g. `15G` = go to page 15)
- `key1 key2` — sequential keys (1 second window)
- Uppercase = `Shift` held

### Navigation

| Key | Action |
|---|---|
| `j` | Scroll down |
| `k` | Scroll up |
| `h` | Scroll left (wide/zoom views) |
| `l` | Scroll right |
| `Ctrl+d` | Half-page down |
| `Ctrl+u` | Half-page up |
| `Space` | Page down |
| `Shift+Space` | Page up |
| `J` | Next page (PDF) / next chapter (EPUB) |
| `K` | Previous page (PDF) / previous chapter (EPUB) |
| `]` | Next EPUB chapter |
| `[` | Previous EPUB chapter |
| `gg` | First page / chapter |
| `G` | Last page / chapter |
| `[N]G` | Go to page N |

### View

| Key | Action |
|---|---|
| `+` | Zoom in |
| `-` | Zoom out |
| `=` | Fit to width |
| `0` | Reset zoom to 100% |
| `d` | Cycle theme: dark → light → sepia |
| `f` | Toggle fullscreen |
| `r` | Rotate page clockwise (PDF) |

### Marks

| Key | Action |
|---|---|
| `m[a-z]` | Set local mark (document-scoped) |
| `m[A-Z]` | Set global mark (cross-document) |
| `` `[a-z] `` | Jump to local mark |
| `` `[A-Z] `` | Jump to global mark (opens document if needed) |
| `''` | Jump to position before last mark jump |

### Bookmarks

| Key | Action |
|---|---|
| `b` | Add bookmark (prompts for label) |
| `B` | Open bookmark list |
| `x` (in list) | Delete selected bookmark |
| `e` (in list) | Edit bookmark label |

### Highlights

| Key | Action | Requires |
|---|---|---|
| `h` | Highlight yellow | Text selected |
| `h r` | Highlight red | Text selected |
| `h g` | Highlight green | Text selected |
| `h b` | Highlight blue | Text selected |
| `H` | Open highlight list | — |
| `n` (in list) | Edit note on highlight | — |
| `x` (in list) | Delete highlight | — |

### Overlays

| Key | Action |
|---|---|
| `o` | Open library |
| `t` | Open table of contents |
| `/` | In-document search |
| `?` | Cross-library annotation search |
| `:` | Command palette |
| `q` | Close viewer, return to library |
| `Escape` | Close top overlay / cancel sequence |

### Command Palette (`:`)

| Command | Action |
|---|---|
| `:goto [N]` | Go to page N |
| `:set theme dark\|light\|sepia` | Change theme |
| `:set font serif\|sans` | EPUB body font |
| `:export highlights` | Export as Markdown |
| `:keys` | Open keyboard reference |

All keybindings are configurable post-MVP via `keys.yaml` in the data directory.

---

## 10. Feature Roadmap

### Phase 1 — MVP

*Goal: everything needed to read, mark, and annotate. Ship this.*

- [ ] `folio init / serve / passwd` CLI
- [ ] bcrypt auth + session cookie
- [ ] Upload PDF + EPUB, SHA256 dedup
- [ ] Metadata extraction (PDF: pdfcpu, EPUB: content.opf)
- [ ] PDF rendering via PDF.js
- [ ] EPUB server-side extraction + chapter serving
- [ ] Keyboard dispatcher (FSM, sequences, numeric prefix)
- [ ] Reading position persistence (auto-save every 5s)
- [ ] Vim-style marks — local (`a–z`) + global (`A–Z`)
- [ ] Highlights — four colors, stored per page/chapter
- [ ] Bookmarks with text labels
- [ ] Library overlay (fuzzy search with fuse.js)
- [ ] Table of contents overlay (PDF outline + EPUB nav)
- [ ] Bookmark + highlight list overlays
- [ ] Status bar (page, marks count, highlights count, filename)
- [ ] Dark / light / sepia themes
- [ ] Docker image + `docker-compose.yml`

### Phase 2 — Polish

*Goal: complete the desktop experience.*

- [ ] Command palette (`:` mode)
- [ ] Configurable keybindings (`keys.yaml`)
- [ ] Highlight notes
- [ ] Export highlights as Markdown / JSON
- [ ] In-document text search (PDF.js find API)
- [ ] Annotation search overlay (`?`)
- [ ] Login rate limiting

### Phase 3 — Search & Sync

*Goal: find anything. Stay in sync across devices.*

- [ ] Full-text document indexing (SQLite FTS5, background job)
- [ ] Cross-library full-text search
- [ ] EPUB CFI for robust highlight anchoring (replaces xpath/offset)
- [ ] Multi-device position sync (polling every 30s on `/api/positions/:id`)
- [ ] Keyboard shortcut reference page (`:keys`)

---

## 11. Deployment

### Docker (recommended)

```yaml
# docker-compose.yml
services:
  folio:
    image: ghcr.io/costabot/folio:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
```

```bash
# First run
docker compose run --rm folio init --data /data --port 8080

# Start
docker compose up -d
```

The `folio init` command is run once outside the server process. The generated password is printed to stdout during that run. After that, `folio serve` starts cleanly from `config.yaml`.

### Build from source

```bash
git clone https://github.com/costabot/folio
cd folio
go build -o folio ./cmd/folio

./folio init --data ./data
./folio serve --data ./data
```

No CGo, no native dependencies. Cross-compiles cleanly:

```bash
GOOS=linux GOARCH=amd64 go build -o folio-linux-amd64 ./cmd/folio
```

### Cloudflare Tunnel

Add to your existing tunnel config on the ThinkPad:

```yaml
# ~/.cloudflared/config.yml
ingress:
  - hostname: folio.yourdomain.com
    service: http://localhost:8080
  # ... other services
  - service: http_status:404
```

Folio's mandatory auth means it's safe to expose directly through the tunnel without an additional proxy layer.

### Environment variable overrides

Config lives in `config.yaml` but all fields can be overridden at runtime:

| Variable | Overrides |
|---|---|
| `FOLIO_PORT` | `port` |
| `FOLIO_DATA_DIR` | `data_dir` |
| `FOLIO_MAX_UPLOAD_MB` | `max_upload_mb` |
| `FOLIO_LOG_LEVEL` | `log_level` |

`password_hash` cannot be overridden by env var — change it only via `folio passwd`.
