#!/bin/bash
APP_DIR="/app"

echo "NUKLAI_RPC=$NUKLAI_RPC" >> ${APP_DIR}/.env
echo "PRIVATE_KEY_BYTES=$PRIVATE_KEY_BYTES" >> ${APP_DIR}/.env
echo "ADMIN_TOKEN=$ADMIN_TOKEN" >> ${APP_DIR}/.env

echo "${@}" | xargs -I % sh -c '%'