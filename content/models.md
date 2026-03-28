---
section: reference
---
# Models

Models are the single source of truth. From a model declaration, Kilnx generates the database table, server validation, and client-side validation attributes.

## Defining a Model

```kilnx
model user
  name: text required min 2 max 100
  email: email unique
  role: option [admin, editor, viewer] default viewer
  active: bool default true
  created: timestamp auto
```

Running `kilnx migrate` creates the SQLite table. `validate user` checks all constraints server-side.

## Field Types

| Type | SQLite column | Description |
|------|--------------|-------------|
| `text` | TEXT | Plain text |
| `email` | TEXT | Email with format validation |
| `int` | INTEGER | Integer number |
| `float` | REAL | Floating point number |
| `bool` | INTEGER (0/1) | Boolean |
| `timestamp` | TEXT (ISO 8601) | Date and time |
| `richtext` | TEXT | Long-form text (HTML allowed) |
| `option` | TEXT | Enum with predefined values |
| `password` | TEXT | Hashed with bcrypt on insert |
| `image` | TEXT | File path for uploads |
| `phone` | TEXT | Phone number |

## Constraints

| Constraint | Description |
|-----------|-------------|
| `required` | Field cannot be empty |
| `unique` | Value must be unique in the table |
| `default <value>` | Default value if not provided |
| `auto` | Auto-generated (timestamps) |
| `min <n>` | Minimum length (text) or value (numbers) |
| `max <n>` | Maximum length (text) or value (numbers) |

## Relationships

Reference another model by using its name as the field type:

```kilnx
model post
  title: text required min 5
  body: richtext required
  author: user required
  created: timestamp auto
```

This creates a column `author_id` as a foreign key to the `user` table. In queries, use `author_id` for the column and `author.name` for JOINed fields.

## Auto-generated Columns

Every model gets an `id` column (INTEGER PRIMARY KEY AUTOINCREMENT) and a `created` column if declared with `auto`.
