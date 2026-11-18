FROM node:22-alpine AS build
WORKDIR /app
# next.config.mjs bakes the rewrite target at build time.
ARG GATEWAY_URL=http://gateway:8080
ENV GATEWAY_URL=$GATEWAY_URL NEXT_TELEMETRY_DISABLED=1
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
ARG GATEWAY_URL=http://gateway:8080
ENV GATEWAY_URL=$GATEWAY_URL
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
USER node
EXPOSE 3000
CMD ["node", "server.js"]
