# syntax=docker/dockerfile:1
#
# taskd web frontend (Next.js + Bun, standalone output). Built independently
# from the Go backend. Requires bun.lock -- run `bun install` locally first to
# generate it, then this image builds reproducibly with --frozen-lockfile.

# --- deps ---
FROM oven/bun:1 AS deps
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

# --- build ---
FROM oven/bun:1 AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN bun run build

# --- runtime ---
FROM gcr.io/distroless/nodejs22-debian12 AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=8080
ENV HOSTNAME=0.0.0.0
# TASKD_API_BASE must point at the taskd backend, e.g. http://taskd:8080
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 8080
CMD ["server.js"]
