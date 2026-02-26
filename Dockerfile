# PicoClaw (tinyland-inc/picoclaw) — standalone Dockerfile
#
# Builds the PicoClaw-based agent with RemoteJuggler config:
# - Multi-stage Go build
# - Bakes in a config.json with Aperture API routing
# - Health check on /health port 18790
#
# Build context: repo root
# GHCR workflow builds from main branch pushes.

# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

RUN addgroup -g 1000 picoclaw && \
    adduser -D -u 1000 -G picoclaw picoclaw && \
    mkdir -p /workspace && chown picoclaw:picoclaw /workspace

USER picoclaw

# Run onboard to create initial directories and config
RUN /usr/local/bin/picoclaw onboard

# --- tinyland customizations ---

# Bake config template with Aperture API routing placeholders.
# entrypoint.sh substitutes ANTHROPIC_API_KEY and ANTHROPIC_BASE_URL at startup.
COPY --chown=picoclaw:picoclaw tinyland/config.json /home/picoclaw/.picoclaw/config.json
COPY --chown=picoclaw:picoclaw tinyland/entrypoint.sh /usr/local/bin/entrypoint.sh

# Workspace bootstrap files — copied to /workspace-defaults/ so the K8s init
# container can seed the PVC on first boot without overwriting evolved state.
COPY --chown=picoclaw:picoclaw tinyland/workspace/ /workspace-defaults/

ENTRYPOINT ["entrypoint.sh"]
CMD ["gateway"]
