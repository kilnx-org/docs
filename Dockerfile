# Build stage: seed docs + compile kilnx app
FROM ghcr.io/kilnx-org/kilnx:0.1.0 AS builder

RUN apk add --no-cache sqlite cmark-gfm

# Seed markdown content into SQLite
COPY content/ /kilnx/content/
COPY seed.sh /kilnx/seed.sh
RUN mkdir -p /data && sh /kilnx/seed.sh /data/docs.db

# Compile the kilnx app
COPY app.kilnx /kilnx/app.kilnx
RUN kilnx build app.kilnx -o /app/server

# Runtime: binary + pre-seeded database
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/server /usr/local/bin/server
COPY --from=builder /data/docs.db /data/docs.db

WORKDIR /data
EXPOSE 8080

CMD ["server"]
