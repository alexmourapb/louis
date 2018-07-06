FROM ubuntu:16.04 as builder
MAINTAINER Alik Khilazhev <alikhil@mail.ru>

ENV LIBVIPS_VERSION 8.6.3

RUN \
  # Install dependencies
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive apt-get install -y \
  automake build-essential curl \
  gobject-introspection gtk-doc-tools libglib2.0-dev libjpeg-turbo8-dev libpng12-dev \
  libwebp-dev libtiff5-dev libgif-dev libexif-dev libxml2-dev libpoppler-glib-dev \
  swig libmagickwand-dev libpango1.0-dev libmatio-dev libopenslide-dev libcfitsio3-dev \
  libgsf-1-dev fftw3-dev liborc-0.4-dev librsvg2-dev && \
  # Build libvips
  cd /tmp && \
  curl -OL https://github.com/jcupitt/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.gz && \
  tar zvxf vips-$LIBVIPS_VERSION.tar.gz && \
  cd /tmp/vips-$LIBVIPS_VERSION && \
  ./configure --enable-debug=no --without-python $1 && \
  make && \
  make install && \
  ldconfig && \
  # Clean up
  apt-get remove -y curl automake build-essential && \
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Go version to use
ENV GOLANG_VERSION 1.10


# gcc for cgo
RUN apt-get update && apt-get install -y \
    gcc curl git libc6-dev make \
    --no-install-recommends \
  && rm -rf /var/lib/apt/lists/*

ENV GOLANG_DOWNLOAD_URL https://golang.org/dl/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 b5a64335f1490277b585832d1f6c7f8c6c11206cba5cd3f771dcb87b98ad1a33

RUN curl -fsSL --insecure "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
  && echo "$GOLANG_DOWNLOAD_SHA256 golang.tar.gz" | sha256sum -c - \
  && tar -C /usr/local -xzf golang.tar.gz \
  && rm golang.tar.gz

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"
WORKDIR $GOPATH

COPY . $GOPATH/src/github.com/KazanExpress/louis
# RUN go get ./...
RUN \
  go get -u gopkg.in/h2non/bimg.v1 && \ 
  go get -u github.com/mattn/go-sqlite3 && \
  go get -u github.com/aws/aws-sdk-go && \
  go get -u github.com/joho/godotenv && \
  go get -u github.com/onsi/gomega && \
  go get -u github.com/stretchr/testify && \
  go get -u github.com/RichardKnop/machinery && \
  go get -u github.com/lib/pq 

RUN go build -o bin/louis github.com/KazanExpress/louis/cmd/louis


FROM ubuntu:16.04

RUN \
  # Install runtime dependencies
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive apt-get install --no-install-recommends -y \
  libglib2.0-0 libjpeg-turbo8 libpng12-0 libopenexr22 \
  libwebp5 libtiff5 libgif7 libexif12 libxml2 libpoppler-glib8 \
  libmagickwand-6.q16-2 libpango1.0-0 libmatio2 libopenslide0 \
  libgsf-1-114 fftw3 liborc-0.4 librsvg2-2 libcfitsio2 && \
  # Clean up
  apt-get autoremove -y && \
  apt-get autoclean && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=builder /usr/local/lib /usr/local/lib
RUN ldconfig
COPY --from=builder /go/bin/louis bin/
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Server port to listen
# unused
ENV PORT 8000 
RUN mkdir /configs
COPY ./cmd/louis/ensure-transforms.json /configs/ensure-transforms.json
# Run the entrypoint command by default when the container starts.
CMD ["bin/louis", "--env=/configs/.env" ,"--transforms-path=/configs/ensure-transforms.json"]
# Expose the server TCP port
EXPOSE 8000
