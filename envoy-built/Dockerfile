FROM --platform=linux/amd64 ubuntu:22.04

# RUN apt-get update && apt-get install -y --no-install-recommends \
#     ca-certificates \
#     && rm -rf /var/lib/apt/lists/*

COPY envoy-static-no-fix /usr/local/bin/envoy

RUN chmod +x /usr/local/bin/envoy

EXPOSE 10000 9901

CMD ["/usr/local/bin/envoy"]
