FROM scratch

COPY ./bin/s3static /s3static

ENTRYPOINT ["/s3static"]
