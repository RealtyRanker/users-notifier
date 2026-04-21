#!/bin/bash
set -e

NETWORK="realty-net"
APP_IMAGE="users-notifier"
APP_CONTAINER="users-notifier"
LOG_DIR="/tmp/users-notifier-logs"

echo "==> Building image: $APP_IMAGE"
docker build -t "$APP_IMAGE" .

echo "==> Stopping existing container (if any)"
docker rm -f "$APP_CONTAINER" 2>/dev/null || true

echo "==> Creating log directory: $LOG_DIR"
mkdir -p "$LOG_DIR"

echo "==> Starting container: $APP_CONTAINER"
docker run -d \
  --name "$APP_CONTAINER" \
  --network "$NETWORK" \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 9091:9091 \
  -v "$(pwd)/config.yaml:/app/config.yaml:ro" \
  -v "$LOG_DIR:/var/log/users-notifier" \
  "$APP_IMAGE"

echo ""
echo "Useful commands:"
echo "  Logs:    docker logs -f $APP_CONTAINER"
echo "  Send:    curl -X POST http://localhost:8080/send -H 'Content-Type: application/json' -d '{\"chat_id\": 123, \"text\": \"hello\"}'"
echo "  Metrics: curl http://localhost:9091/metrics"
echo "  Health:  curl http://localhost:9091/healthz"
echo "  Stop:    docker stop $APP_CONTAINER"