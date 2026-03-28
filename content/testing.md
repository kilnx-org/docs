---
section: reference
---
# Testing and i18n

## Declarative Tests

Test your app in the same language. No Selenium, no Cypress, no test framework.

```kilnx
test "user can register"
  visit /register
  fill name with "Alice"
  fill identity with "alice@test.com"
  fill password with "secret123"
  submit
  expect page /login contains "Log in"

test "homepage loads"
  visit /
  expect status 200
  expect page / contains "Blog"

test "post requires auth"
  visit /posts/new
  expect page /login
```

Run tests:

```bash
$ kilnx test app.kilnx
Running 3 test(s):
  PASS  user can register
  PASS  homepage loads
  PASS  post requires auth
All 3 test(s) passed.
```

Tests run against a real in-memory server with a fresh SQLite database. No mocks.

## Internationalization

```kilnx
translations
  en
    welcome: "Welcome back"
    users: "Users"
  pt
    welcome: "Bem vindo de volta"
    users: "Usuários"

config
  default language: en
  detect language: header accept-language
```

Use `{t.key}` in pages:

```kilnx
page /dashboard requires auth
  "{t.welcome}, {current_user.name}"
```

Language is detected from:
1. `?lang=pt` query parameter (highest priority)
2. `Accept-Language` HTTP header
3. Default language from config
