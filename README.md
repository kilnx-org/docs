<p align="center">
  <a href="https://kilnx.dev"><img src="https://raw.githubusercontent.com/kilnx-org/kilnx/main/.github/banner.svg" alt="kilnx" width="600"/></a>
</p>

## Kilnx Documentation

The official Kilnx documentation site, built with Kilnx.

**Live**: [docs.kilnx.dev](https://docs.kilnx.dev)

### How it works

Documentation is written as markdown files in `content/`. A seed script converts them to SQLite at build time. A Kilnx app serves the pages with search (htmx), feedback voting, and a dark-themed layout matching the main site.

```
content/*.md → seed.sh → SQLite → app.kilnx serves pages
```

### Contributing docs

1. Edit or create a `.md` file in `content/`
2. Add frontmatter with `section: guide` or `section: reference`
3. Open a PR
4. Merge triggers automatic rebuild on Railway

### Running locally

```bash
# Seed the database
sh seed.sh docs.db

# Run the docs site
kilnx run app.kilnx
```

### Structure

```
content/
  getting-started.md    Guide: install, hello world, CLI
  models.md             Reference: field types, constraints, relationships
  pages.md              Reference: pages, actions, fragments, templates
  auth.md               Reference: auth, permissions, security
  realtime.md           Reference: SSE, WebSockets, broadcast
  api.md                Reference: JSON API, CORS
  webhooks.md           Reference: webhooks, jobs, schedules, rate limiting
  testing.md            Reference: declarative tests, i18n
  deploy.md             Guide: binary, Railway, Docker, Fly.io, systemd
  principles.md         Guide: design principles
app.kilnx               The doc engine
seed.sh                 Markdown → SQLite loader
Dockerfile              Build pipeline
```
