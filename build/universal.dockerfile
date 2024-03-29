FROM kexpress/louis-deps:1.12.4 as builder
LABEL Author="Alik Khilazhev <alikhil@mail.ru>"

RUN  mkdir -p /go/src/github.com/KazanExpress/louis

COPY . /go/src/github.com/KazanExpress/louis

ARG APP=louis

RUN go build -o bin/${APP} github.com/KazanExpress/louis/cmd/${APP}

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
COPY --from=builder /go/bin/${APP} bin/
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# Server port to listen
# unused
ENV PORT 8000 
RUN mkdir /configs 
# Expose the server TCP port
EXPOSE 8000