---
section: reference
---
# Realtime (SSE and WebSockets)

## Server-Sent Events

`stream` creates an SSE endpoint with automatic polling.

```kilnx
stream /notifications requires auth
  query: SELECT message FROM notification
         WHERE user_id = :current_user.id AND seen = false
  every 5s
```

The htmx SSE extension is embedded in the binary. Use it in pages:

```html
<div hx-ext="sse" sse-connect="/notifications" sse-swap="message">
  Waiting for notifications...
</div>
```

## WebSockets

`socket` creates a bidirectional WebSocket with rooms and broadcast.

```kilnx
socket /chat/:room requires auth
  on connect
    query: SELECT message, author FROM chat_message
           WHERE room = :room ORDER BY created DESC LIMIT 50
    send history

  on message
    validate
      body: required max 500
    query: INSERT INTO chat_message (body, author_id, room)
           VALUES (:body, :current_user.id, :room)
    broadcast to :room fragment chat-bubble
```

`on connect` runs when a client connects. `on message` runs when the client sends data. `broadcast to :room` sends to all connected clients in that room.

## Broadcast

`broadcast` sends a fragment to all WebSocket clients in a room:

```kilnx
broadcast to :room fragment chat-bubble
```

The fragment receives the same parameters that the socket handler received.
