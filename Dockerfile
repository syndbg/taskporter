FROM scratch

COPY taskporter /taskporter

ENTRYPOINT ["/taskporter"]
