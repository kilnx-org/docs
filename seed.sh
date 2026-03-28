#!/bin/sh
# Seed documentation content from markdown files into SQLite
# Runs during Docker build to populate the database

set -e

DB="${1:-/data/docs.db}"

# Create tables
sqlite3 "$DB" <<'SQL'
CREATE TABLE IF NOT EXISTS doc (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  slug TEXT UNIQUE NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  section TEXT NOT NULL DEFAULT 'guide',
  sort_order INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS vote (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_slug TEXT NOT NULL,
  helpful INTEGER NOT NULL DEFAULT 1,
  created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS doc_fts USING fts5(
  title, body, slug, content='doc', content_rowid='id'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER IF NOT EXISTS doc_ai AFTER INSERT ON doc BEGIN
  INSERT INTO doc_fts(rowid, title, body, slug) VALUES (new.id, new.title, new.body, new.slug);
END;

CREATE TRIGGER IF NOT EXISTS doc_ad AFTER DELETE ON doc BEGIN
  INSERT INTO doc_fts(doc_fts, rowid, title, body, slug) VALUES ('delete', old.id, old.title, old.body, old.slug);
END;

CREATE TRIGGER IF NOT EXISTS doc_au AFTER UPDATE ON doc BEGIN
  INSERT INTO doc_fts(doc_fts, rowid, title, body, slug) VALUES ('delete', old.id, old.title, old.body, old.slug);
  INSERT INTO doc_fts(rowid, title, body, slug) VALUES (new.id, new.title, new.body, new.slug);
END;
SQL

echo "Database initialized: $DB"

# Insert/update each markdown file
ORDER=0
for file in content/*.md; do
  [ -f "$file" ] || continue

  SLUG=$(basename "$file" .md)

  # Extract title from first # heading
  TITLE=$(head -5 "$file" | grep '^# ' | head -1 | sed 's/^# //')
  [ -z "$TITLE" ] && TITLE="$SLUG"

  # Extract section from frontmatter (line starting with "section:")
  SECTION=$(grep '^section:' "$file" 2>/dev/null | head -1 | sed 's/^section: *//')
  [ -z "$SECTION" ] && SECTION="guide"

  # Read body: strip frontmatter, convert markdown to HTML
  BODY=$(awk '
    BEGIN { in_front=0; started=0 }
    /^---$/ && !started { in_front=1; started=1; next }
    /^---$/ && in_front { in_front=0; next }
    !in_front { print }
  ' "$file" | cmark --unsafe)

  ORDER=$((ORDER + 1))

  sqlite3 "$DB" "INSERT OR REPLACE INTO doc (slug, title, body, section, sort_order, updated_at)
    VALUES ('$SLUG',
            '$(echo "$TITLE" | sed "s/'/''/g")',
            '$(echo "$BODY" | sed "s/'/''/g")',
            '$SECTION',
            $ORDER,
            datetime('now'));"

  echo "  Loaded: $SLUG ($TITLE)"
done

echo "Seeded $(sqlite3 "$DB" "SELECT count(*) FROM doc") docs"
