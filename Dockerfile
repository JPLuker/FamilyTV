FROM node:22-alpine AS webbuild
WORKDIR /web
COPY web/package*.json ./
RUN npm ci --no-audit --no-fund
COPY web/ ./
RUN npm run build
FROM golang:1.24-alpine AS gobuild
WORKDIR /src
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /familytv ./cmd/server
FROM alpine:3.21
RUN addgroup -S familytv && adduser -S -G familytv familytv && mkdir -p /data && chown familytv:familytv /data
COPY --from=gobuild /familytv /usr/local/bin/familytv
COPY --from=webbuild /web/dist /opt/familytv/web
USER familytv
ENV FAMILYTV_LISTEN=0.0.0.0:8080 FAMILYTV_STATE_PATH=/data/state.json FAMILYTV_WEB_DIR=/opt/familytv/web
EXPOSE 8080
CMD ["/usr/local/bin/familytv"]
