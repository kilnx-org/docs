#!/bin/sh
# Convert markdown to HTML with GFM table support
# Uses cmark for most conversion, then post-processes pipe tables
# Usage: cat file.md | sh md2html.sh

# First pass: convert pipe tables to HTML before cmark
# Second pass: run cmark on the result

awk '
BEGIN { in_table = 0 }

# Detect table: line with | separators
/^\|.*\|$/ {
  # Check if next-ish line is separator (|---|)
  if (!in_table) {
    in_table = 1
    # Parse header
    gsub(/^\| *| *\|$/, "")
    n = split($0, cells, / *\| */)
    printf "<table>\n<thead><tr>"
    for (i = 1; i <= n; i++) printf "<th>%s</th>", cells[i]
    printf "</tr></thead>\n<tbody>\n"
    next
  }
  # Skip separator line
  if ($0 ~ /^\|[\-\| :]+\|$/) next
  # Data row
  gsub(/^\| *| *\|$/, "")
  n = split($0, cells, / *\| */)
  printf "<tr>"
  for (i = 1; i <= n; i++) printf "<td>%s</td>", cells[i]
  printf "</tr>\n"
  next
}

# End of table
{
  if (in_table) {
    printf "</tbody></table>\n"
    in_table = 0
  }
  print
}

END {
  if (in_table) printf "</tbody></table>\n"
}
' | cmark --unsafe 2>/dev/null || cmark
