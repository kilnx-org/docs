---
section: guide
---
# Design Principles

These principles are constitutional. If any future design decision contradicts a principle, the principle wins.

## 0. The complexity is the tool's fault, not the problem's

Most web apps are not complex. They are lists, forms, dashboards, CRUDs. The complexity comes from the tools we use, not from the problem we are solving. Kilnx exists to prove this.

## 1. Zero decisions before the first useful line

The developer does not choose a framework, does not configure anything, does not install dependencies. They create a file, write business logic, run it.

## 2. SQL is a first-class citizen

SQL is not something you "call" from inside another language. SQL IS part of the language. Queries live inline, not in strings, not in ORMs, not in separate files. The database is the heart, not an accessory.

## 3. HTML is the native output

The language thinks in HTML. Not JSON, not XML, not protobuf. It exists to serve HTML to the browser and to htmx.

## 4. Declarative first, imperative when necessary

You declare what you want, not how to do it. "This page requires auth" instead of 30 lines of middleware.

## 5. One file can be a complete app

Just like a `.html` can be a website and a `.sql` in SQLPage can be a page, a single `.kilnx` file can be a functional web app. Complexity is opt-in, not mandatory.

## 6. The binary is the deploy

Compile, get an executable. Copy to the server, run. No runtime, no mandatory container, no 200MB of node_modules.

## 7. Fragments are first-class

The language natively understands the concept of "a piece of HTML" that htmx will swap in the DOM. Fragments, partials, pieces are the basic unit of response.

## 8. Security is default, not opt-in

CSRF, SQL injection, XSS, session management come solved by default. The developer needs to make effort to be insecure, not to be secure.

## 9. Zero dependencies for the user

The language may use Go, C, or whatever underneath. But the user never sees it. Never installs anything besides the CLI.

## 10. htmx awareness without htmx coupling

The language understands htmx concepts (fragments, swaps, triggers, SSE) and makes it easy to serve these patterns. But it does not depend on htmx. htmx is the natural pair, not a hard dependency.
