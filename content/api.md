---
section: reference
---
# JSON API

`api` works like `page` but returns JSON instead of HTML.

```kilnx
api /api/v1/users requires auth
  query users: SELECT id, name, email FROM user
               ORDER BY id DESC paginate 50
```

Response:

```json
{
  "data": [
    {"id": "1", "name": "Alice", "email": "alice@example.com"},
    {"id": "2", "name": "Bob", "email": "bob@example.com"}
  ],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total": 2
  }
}
```

## Mutation API

```kilnx
api /api/v1/posts method POST requires editor
  validate
    title: required min 5
    body: required
  query: INSERT INTO post (title, body, author_id, status)
         VALUES (:title, :body, :current_user.id, 'draft')
  respond status 201
```

## CORS

Configure allowed origins in the config block:

```kilnx
config
  cors: https://myapp.com, https://staging.myapp.com
```

Without `cors:`, cross-origin requests are blocked (same-origin only).
