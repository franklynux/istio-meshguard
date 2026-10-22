# syntax=docker/dockerfile:1
# Build from the repo root:
#   docker build -f services/notifications/Dockerfile -t notifications:local .
FROM node:24-slim

ENV NODE_ENV=production
WORKDIR /app

# Install dependencies first so this layer is cached until they change.
COPY services/notifications/package.json services/notifications/package-lock.json ./
RUN npm ci --omit=dev

COPY services/notifications/server.js services/notifications/confirmations.js ./
COPY proto/notifications.proto ./proto/notifications.proto
ENV PROTO_PATH=/app/proto/notifications.proto

# The base image's built-in "node" user has UID 1000. Numeric, so Kubernetes
# can verify runAsNonRoot.
USER 1000
EXPOSE 50053
CMD ["node", "server.js"]