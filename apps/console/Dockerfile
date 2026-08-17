# syntax=docker/dockerfile:1
FROM node:22-alpine AS deps
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile || pnpm install

FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
# Baked into the client bundle at build time (runtime env cannot change NEXT_PUBLIC_*).
ARG NEXT_PUBLIC_API_URL=https://api.raisin.run
ARG BETTER_AUTH_URL=https://app.raisin.run
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV BETTER_AUTH_URL=$BETTER_AUTH_URL
RUN pnpm build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN addgroup -S raisin && adduser -S raisin -G raisin
COPY --from=build /app/public ./public
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
USER raisin
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
# Server-side proxy target (overridable at runtime via Helm).
ENV RAISIN_API_URL=http://raisin-api
CMD ["node", "server.js"]
