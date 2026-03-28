---
section: reference
---
# Auth and Permissions

## Auth

Six lines for a complete auth system: registration, login, logout, bcrypt hashing, session cookies, and `current_user` available everywhere.

```kilnx
auth
  table: user
  identity: email
  password: password
  login: /login
  after login: /dashboard
```

This auto-generates:
- `/login` page with email/password form
- `/register` page with name/email/password form
- `/logout` endpoint (POST with CSRF)
- Session cookies (HMAC-signed, HttpOnly, 24h TTL)
- `current_user.id`, `current_user.identity`, `current_user.role` available in queries

## Protecting Routes

```kilnx
page /dashboard requires auth
  "Welcome back, {current_user.identity}"

page /admin requires admin
  "Admin panel"

action /users/delete method POST requires admin
  query: DELETE FROM user WHERE id = :id
```

`requires auth` redirects to login if not authenticated. `requires <role>` checks the user's role.

## Permissions

Define role-based access rules:

```kilnx
permissions
  admin: all
  editor: read post, write post where author = current_user
  viewer: read post where status = published
```

## Security Defaults

All of these are automatic, no configuration needed:

- Passwords hashed with bcrypt (cost 10)
- Session IDs signed with HMAC-SHA256
- CSRF tokens on all forms (auto-injected via `{csrf}`)
- SQL parameters bound (never interpolated)
- HTML output escaped by default (`| raw` to opt out)
- Session cleanup every 5 minutes
- Login redirect validated against open redirect attacks
