FROM golang:1.22.3-bookworm as dev

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV PACKAGES="ca-certificates git curl bash zsh"
ENV ROOT /app

RUN apt-get update && apt-get install -y ${PACKAGES} && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR ${ROOT}

COPY ./ ./

RUN go mod download

EXPOSE 9478

CMD [ "go", "run", "main.go" ]

# ---
FROM golang:1.22.3-bookworm as builder

ENV CGO_ENABLED=0
ENV GO111MODULE=on
ENV ROOT /app
ENV OUT_DIR ${ROOT}/out

RUN apt-get update && apt-get install -y ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

WORKDIR ${ROOT}

COPY ./ ./

RUN go mod download && go build -o ${OUT_DIR} ${ROOT}/main.go

# ---
FROM debian:12.5-slim as prod

ENV ROOT /app
ENV OUT_DIR ${ROOT}/out

USER nobody

WORKDIR ${ROOT}
COPY --from=builder --chown=nobody:nogroup ${OUT_DIR} .

RUN apt-get update && apt-get install -y curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

CMD [ "sentinel-exporter" ]
