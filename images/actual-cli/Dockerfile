FROM node:24.18.0-bookworm-slim

ARG ACTUAL_CLI_VERSION=26.7.0

RUN npm install --global "@actual-app/cli@${ACTUAL_CLI_VERSION}" \
    && npm cache clean --force

USER node

ENV ACTUAL_DATA_DIR=/tmp/actual-cli

ENTRYPOINT ["actual"]
CMD ["--help"]
