# ─── Build stage ─────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies first (layer cache optimization)
COPY package.json package-lock.json* ./
RUN npm ci

# Build the SPA
COPY . .
RUN npm run build

# ─── Serve stage — static files behind nginx ─────────────────
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

# 127.0.0.1, not localhost: nginx listens on IPv4 only, while localhost
# resolves to ::1 first inside the container and the probe would be refused.
HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget -qO- http://127.0.0.1/ >/dev/null || exit 1
