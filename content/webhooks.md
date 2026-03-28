---
section: reference
---
# Webhooks, Jobs, and Schedules

## Webhooks

Receive external events with HMAC signature verification.

```kilnx
webhook /stripe/payment secret env STRIPE_SECRET
  on event payment_intent.succeeded
    query: UPDATE order SET status = 'paid'
           WHERE stripe_id = :event_id
    send email to :event_customer_email
      subject: "Payment confirmed"
```

Use `on event *` as a catch-all:

```kilnx
webhook /github secret env GITHUB_SECRET
  on event *
    query: INSERT INTO webhook_log (event, payload)
           VALUES (:event.type, :event.body)
```

## Background Jobs

Define async work with `job` and dispatch with `enqueue`:

```kilnx
job send-welcome
  query data: SELECT name, email FROM user WHERE id = :user_id
  send email to :email
    subject: "Welcome {data.name}"

action /users/create method POST
  query: INSERT INTO user (name, email) VALUES (:name, :email)
  enqueue send-welcome with user_id: :id
  redirect /users
```

## Schedules

Timed tasks running inside the same binary. No Redis, no cron.

```kilnx
schedule cleanup every 24h
  query: DELETE FROM session WHERE expires_at < datetime('now')

schedule report every monday at 9:00
  query stats: SELECT count(*) as new_users FROM user
               WHERE created > datetime('now', '-7 days')
  send email to "admin@example.com"
    subject: "Weekly report: {stats.new_users} new users"
```

## Rate Limiting

```kilnx
limit /api/*
  requests: 100 per minute per user

limit /login
  requests: 5 per minute per ip
  on exceeded: status 429 message "Too many attempts"
```
