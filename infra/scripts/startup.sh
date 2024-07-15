#!/bin/bash
APP_DIR="/app"

echo "PRIVATE_KEY_BYTES="$PRIVATE_KEY_BYTES"" >> ${APP_DIR}/.env
echo "NUKLAI_RPC="$NUKLAI_RPC"" >> ${APP_DIR}/.env
echo "ADMIN_TOKEN="$ADMIN_TOKEN"" >> ${APP_DIR}/.env
echo "POSTGRES_HOST="$POSTGRES_HOST"" >> ${APP_DIR}/.env
echo "POSTGRES_PORT="$POSTGRES_PORT"" >> ${APP_DIR}/.env
echo "POSTGRES_USER="$POSTGRES_USER"" >> ${APP_DIR}/.env
echo "POSTGRES_PASSWORD="$POSTGRES_PASSWORD"" >> ${APP_DIR}/.env
echo "POSTGRES_DBNAME="$POSTGRES_DBNAME"" >> ${APP_DIR}/.env
echo "POSTGRES_ENABLESSL="$POSTGRES_ENABLESSL"" >> ${APP_DIR}/.env

#echo "${@}" | xargs -I % sh -c '%'