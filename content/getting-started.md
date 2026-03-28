---
section: guide
---
# Getting Started

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/kilnx-org/kilnx/main/install.sh | sh
```

Or build from source (requires Go 1.24+):

```bash
git clone https://github.com/kilnx-org/kilnx.git
cd kilnx && go build -o kilnx ./cmd/kilnx/
sudo mv kilnx /usr/local/bin/
```

## Hello World

Create a file called `app.kilnx`:

```kilnx
page /
  "Hello World"
```

Run it:

```bash
kilnx run app.kilnx
```

Open `http://localhost:8080`. That's it. One useful line, a running web server with htmx linked automatically.

## A Task List in 30 Lines

```kilnx
model task
  title: text required
  done: bool default false
  created: timestamp auto

auth
  table: user
  identity: email
  password: password
  login: /login
  after login: /tasks

page /tasks requires auth
  query tasks: SELECT id, title, done FROM task
               WHERE owner_id = :current_user.id
               ORDER BY created DESC paginate 20
  html
    {{each tasks}}
    <tr>
      <td>{title}</td>
      <td>{{if done}}Yes{{end}}</td>
      <td><button hx-post="/tasks/{id}/delete"
                  hx-target="closest tr" hx-swap="outerHTML">Delete</button></td>
    </tr>
    {{end}}

action /tasks/new requires auth
  validate task
  query: INSERT INTO task (title, owner_id) VALUES (:title, :current_user.id)
  redirect /tasks
```

This gives you: registration, login with bcrypt, sessions, CSRF, paginated search, validation, inline htmx delete, and a SQLite database.

## CLI Commands

| Command | Description |
|---------|-------------|
| `kilnx run <file>` | Dev server with hot reload |
| `kilnx build <file> -o app` | Compile to standalone binary (~15MB) |
| `kilnx check <file>` | Static analysis and security scan |
| `kilnx test <file>` | Run declarative tests |
| `kilnx migrate <file>` | Apply database migrations |
| `kilnx lsp` | Start Language Server Protocol server |

## Deploy

Compile and copy:

```bash
kilnx build app.kilnx -o myapp
scp myapp server:~/
ssh server './myapp'
```

Or deploy on Railway with one click using the [Kilnx Railway template](https://railway.com/deploy/kilnx).
