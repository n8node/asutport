FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install

FROM node:22-alpine AS builder
WORKDIR /app
ARG BUILD_ID
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN EFFECTIVE_BUILD_ID="${BUILD_ID:-$(date -u +%Y%m%d%H%M%S)}" \
  && BUILD_ID="$EFFECTIVE_BUILD_ID" NEXT_PUBLIC_BUILD_ID="$EFFECTIVE_BUILD_ID" npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
