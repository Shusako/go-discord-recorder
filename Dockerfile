# https://hub.docker.com/r/nvidia/cuda/tags?page_size=&ordering=&name=devel-ubuntu
# must match the cude toolkit version you have installed
# also update the runner below
FROM nvidia/cuda:12.6.1-devel-ubuntu24.04 AS build
RUN apt update \
    && apt install -y wget build-essential \
    && apt clean

# Install Go tools
WORKDIR /tmp
# Copy linux link from https://go.dev/dl/ to update
RUN wget https://go.dev/dl/go1.23.1.linux-amd64.tar.gz && \
    tar -xzf go1.23.1.linux-amd64.tar.gz
ENV PATH="/tmp/go/bin:${PATH}"
ENV GOPATH="/tmp/go"

# Build whisper shared library
WORKDIR /build/whisper.cpp
COPY whisper.cpp/ .
RUN GGML_CUDA=1 make libwhisper.so -j

WORKDIR /build
# Download dependencies first
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build the rest
COPY . .
ARG CGO_ENABLED=1
ARG CGO_CFLAGS=-I/build/whisper.cpp -I/usr/local/cuda/include -I/opt/cuda/include -I/targets/x86_64-linux/include
ARG CGO_CXXFLAGS=-I/build/whisper.cpp -I/usr/local/cuda/include -I/opt/cuda/include -I/targets/x86_64-linux/include
ARG CGO_LDFLAGS=-L/build/whisper.cpp -lcublas -lculibos -lcudart -lcublasLt -lpthread -ldl -lrt -L/usr/local/cuda/lib64 -L/opt/cuda/lib64 -L/targets/x86_64-linux/lib -L/usr/local/nvidia/lib -L/usr/local/nvidia/lib64

# Define a build argument
ARG BUILD_RACE

# Conditional build command
RUN if [ "$BUILD_RACE" = "1" ]; then \
        go build -race -o go-discord-recorder; \
    else \
        go build -o go-discord-recorder; \
    fi

# Prepare runner image
FROM nvidia/cuda:12.6.1-devel-ubuntu24.04 AS runner

RUN apt update \
    && apt install -y ffmpeg \
    && apt clean

WORKDIR /
COPY --from=build /build/resources /resources
COPY --from=build /build/go-discord-recorder /go-discord-recorder
COPY --from=build /build/whisper.cpp/libwhisper.so /usr/lib/x86_64-linux-gnu/libwhisper.so
RUN ldconfig

CMD ["/go-discord-recorder"]