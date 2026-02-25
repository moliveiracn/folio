# folio

> A self-hosted, keyboard-driven document reader for PDF and EPUB.  
> Sioyek spirit. Browser delivery. Your data, your machine.

**Status: pre-development — spec in progress, no code yet**

Folio inverts the usual self-hosted reading tools: the reader is the product, and the library is just a doorway into it. It brings [sioyek](https://github.com/ahrm/sioyek)'s keyboard-first, research-oriented experience to the browser — self-hosted, single-user, accessible from any device on your network.

---

## Planned features

- **PDF and EPUB** — both formats, one reader
- **Keyboard-first** — vim-style marks, multi-key sequences, numeric prefixes
- **Highlights** in four colors with optional notes
- **Bookmarks** with text labels, searchable across your library
- **Vim marks** — lowercase local to document, uppercase global across library
- **Table of contents** — PDF outline and EPUB nav, fuzzy searchable
- **Reading position** — auto-saved, always resumes where you left off
- **Single binary** — one `folio` executable, one `folio.db`, nothing else
- **Mandatory auth** — bcrypt password, safe to expose via Cloudflare Tunnel

---

## Roadmap

### Phase 1 — MVP
*Goal: everything needed to read, mark, and annotate.*

- [ ] `folio init / serve / passwd` CLI
- [ ] bcrypt auth + session cookie — mandatory, code-server style
- [ ] PDF upload + rendering via PDF.js
- [ ] EPUB upload + server-side chapter extraction and serving
- [ ] SHA256 content-addressed storage — uploading the same file twice is a no-op
- [ ] Metadata extraction (PDF info dict, EPUB `content.opf`)
- [ ] Keyboard dispatcher — FSM with multi-key sequences and numeric prefix
- [ ] Reading position persistence — auto-save every 5s, always resumes
- [ ] Vim-style marks — local (`a–z`) and global (`A–Z`)
- [ ] Highlights — four colors, per page (PDF) and per chapter (EPUB)
- [ ] Bookmarks with text labels
- [ ] Library overlay — fuzzy search with fuse.js
- [ ] Table of contents overlay — PDF outline + EPUB nav document
- [ ] Bookmark + highlight list overlays
- [ ] Status bar — page, mark count, highlight count, filename
- [ ] Dark / light / sepia themes
- [ ] Docker image + `docker-compose.yml`

### Phase 2 — Polish
*Goal: complete the desktop experience.*

- [ ] Command palette (`:` mode)
- [ ] Configurable keybindings via `keys.yaml`
- [ ] Highlight notes (free-text annotation attached to a highlight)
- [ ] Export highlights as Markdown or JSON
- [ ] In-document text search (PDF.js find API)
- [ ] Annotation search overlay (`?`)
- [ ] Login rate limiting

### Phase 3 — Search & Sync
*Goal: find anything. Stay in sync across devices.*

- [ ] Full-text document indexing — SQLite FTS5, background job after upload
- [ ] Cross-library full-text search
- [ ] EPUB CFI — replaces xpath/offset for robust highlight anchoring
- [ ] Multi-device position sync — polling every 30s
- [ ] Keyboard shortcut reference overlay (`:keys`)

---

## Documentation

- [`DESIGN.md`](./DESIGN.md) — full architecture, data model, and API spec

---

## License

AGPL-3.0 — see [LICENSE](./LICENSE).
