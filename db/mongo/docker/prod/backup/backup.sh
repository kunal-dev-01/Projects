#!/bin/bash
set -e

echo "-----------------------------------"
echo "Starting MongoDB backup at $(date)"

# Environment variables mapping
MONGO_HOST=${MONGO_HOST:-mongo_db}
MONGO_PORT=${MONGO_PORT:-27017}
# We use the same root credentials passed to the mongo_db container via .env.prod
MONGO_USER=${MONGO_INITDB_ROOT_USERNAME:-$MONGO_USER}
MONGO_PASSWORD=${MONGO_INITDB_ROOT_PASSWORD:-$MONGO_PASSWORD}
S3_BUCKET=${S3_BUCKET}
S3_PREFIX=${S3_PREFIX:-mongo-backups}

TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_NAME="mongo_backup_${TIMESTAMP}"
BACKUP_DIR="/tmp/${BACKUP_NAME}"
ARCHIVE_NAME="${BACKUP_NAME}.tar.gz"

if [ -z "$S3_BUCKET" ]; then
  echo "Error: S3_BUCKET is not set in environment."
  exit 1
fi

if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
  echo "Error: AWS credentials are not set."
  exit 1
fi

mkdir -p "$BACKUP_DIR"

echo "Dumping databases from $MONGO_HOST..."
mongodump --host="$MONGO_HOST" --port="$MONGO_PORT" \
          --username="$MONGO_USER" --password="$MONGO_PASSWORD" \
          --authenticationDatabase=admin \
          --out="$BACKUP_DIR"

echo "Compressing backup..."
cd /tmp
tar -zcvf "$ARCHIVE_NAME" "$BACKUP_NAME"

echo "Uploading to S3..."
# Endpoint URL handles both AWS S3 and custom S3 providers like Cloudflare R2 / MinIO
if [ -n "$AWS_ENDPOINT_URL" ]; then
  aws s3 cp "$ARCHIVE_NAME" "s3://${S3_BUCKET}/${S3_PREFIX}/${ARCHIVE_NAME}" --endpoint-url "$AWS_ENDPOINT_URL"
else
  aws s3 cp "$ARCHIVE_NAME" "s3://${S3_BUCKET}/${S3_PREFIX}/${ARCHIVE_NAME}"
fi

echo "Cleaning up local files..."
rm -rf "$BACKUP_DIR"
rm -f "/tmp/$ARCHIVE_NAME"

echo "Backup completed successfully at $(date)!"
