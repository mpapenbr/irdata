# This file is used by goreleaser
FROM scratch
ENTRYPOINT ["/irdata"]
COPY irdata /
