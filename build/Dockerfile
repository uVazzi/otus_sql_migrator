FROM golang:1.24.3-alpine3.21 AS build

ENV BIN_FILE="/opt/gomigrator/gomigrator"
ENV CODE_DIR=/go/src/

WORKDIR ${CODE_DIR}

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . ${CODE_DIR}
RUN CGO_ENABLED=0 go build \
        -v -o ${BIN_FILE} cmd/gomigrator/*

FROM alpine:3.21

ARG CONFIG
ARG MIGRATION_DIR_PATH

ENV BIN_FILE="/opt/gomigrator/gomigrator"
COPY --from=build ${BIN_FILE} ${BIN_FILE}

ENV CONFIG_FILE="/etc/gomigrator/config.yml"
COPY ${CONFIG} ${CONFIG_FILE}

ENV MIGRATION_DIR='/etc/gomigrator/migrations/'
COPY ${MIGRATION_DIR_PATH} ${MIGRATION_DIR}

ENTRYPOINT ["/opt/gomigrator/gomigrator", "--config=/etc/gomigrator/config.yml", "--dir=/etc/gomigrator/migrations"]
