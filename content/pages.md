---
section: reference
---
# Pages, Actions, and Fragments

## Pages

A `page` is a GET route that returns full HTML.

```kilnx
page /users layout main title "Users"
  query users: SELECT name, email FROM user ORDER BY id DESC
  html
    {{each users}}
    <div class="user">
      <strong>{name}</strong>
      <span>{email}</span>
    </div>
    {{end}}
```

Options: `layout <name>`, `title "text"`, `requires auth`, `requires <role>`.

## Actions

An `action` is a POST/PUT/DELETE route that mutates data.

```kilnx
action /posts/create method POST requires auth
  validate
    title: required min 5
    body: required
  query: INSERT INTO post (title, body, author_id)
         VALUES (:title, :body, :current_user.id)
  on success
    redirect /posts
  on error
    alert "Could not create post"
```

Named parameters (`:title`, `:body`) come from form fields. URL parameters (`:id`) come from the path (`/posts/:id/delete`). `:current_user.id`, `:current_user.identity`, and `:current_user.role` are available when `requires auth` is set.

## Fragments

A `fragment` returns partial HTML for htmx to swap into the DOM.

```kilnx
fragment /users/:id/card
  query user: SELECT name, email FROM user WHERE id = :id
  html
    <div class="card">
      <strong>{user.name}</strong>
      <span>{user.email}</span>
    </div>
```

Use with htmx:

```html
<button hx-get="/users/42/card" hx-target="#detail">Show</button>
```

## Respond with Fragment

Actions can respond with a fragment instead of redirecting:

```kilnx
action /users/:id/delete requires auth
  query: DELETE FROM user WHERE id = :id
  respond fragment delete
```

`respond fragment delete` tells htmx to remove the element. You can also respond with a query:

```kilnx
action /tasks/:id/toggle method POST
  query: UPDATE task SET done = NOT done WHERE id = :id
  respond fragment task-row with query:
    SELECT id, title, done FROM task WHERE id = :id
```

## Template Syntax

Inside `html` blocks:

| Syntax | Description |
|--------|-------------|
| `{field}` | Output a field value (HTML-escaped) |
| `{field \| raw}` | Output without escaping |
| `{field \| truncate: 30}` | Apply a filter |
| `{{each query_name}}...{{end}}` | Loop over query results |
| `{{if field}}...{{end}}` | Conditional rendering |
| `{{if field}}...{{else}}...{{end}}` | If/else |
| `{csrf}` | CSRF token for forms |

## Filters

| Filter | Example | Output |
|--------|---------|--------|
| `upcase` | `{name \| upcase}` | ALICE |
| `truncate` | `{name \| truncate: 20}` | Alice Wonderla... |
| `date` | `{created \| date: "Jan 02, 2006"}` | Mar 27, 2026 |
| `timeago` | `{created \| timeago}` | 3 hours ago |
| `currency` | `{price \| currency: "$"}` | $1,234.56 |
| `pluralize` | `{count \| pluralize: "item", "items"}` | 3 items |
| `raw` | `{bio \| raw}` | Unescaped HTML |
| `default` | `{role \| default: "viewer"}` | viewer (if empty) |
