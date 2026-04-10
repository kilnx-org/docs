# Build stage: seed docs + compile kilnx app + playground gateway
FROM ghcr.io/kilnx-org/kilnx:0.1.0 AS builder

RUN apk add --no-cache sqlite cmark go

# Seed markdown content into SQLite
COPY content/ /kilnx/content/
COPY seed.sh /kilnx/seed.sh
COPY md2html.sh /kilnx/md2html.sh
RUN mkdir -p /data && sh /kilnx/seed.sh /data/docs.db

# Compile the kilnx docs app
COPY app.kilnx /kilnx/app.kilnx
RUN kilnx build app.kilnx -o /app/server

# Compile playground gateway
COPY playground.go /kilnx/playground.go
RUN cd /kilnx && CGO_ENABLED=0 go build -o /app/gateway playground.go

# Runtime: gateway + docs server + kilnx CLI
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/server /usr/local/bin/server
COPY --from=builder /app/gateway /usr/local/bin/gateway
COPY --from=builder /usr/local/bin/kilnx /usr/local/bin/kilnx
COPY --from=builder /data/docs.db /data/docs.db

WORKDIR /data
EXPOSE 8080

CMD ["gateway"]
