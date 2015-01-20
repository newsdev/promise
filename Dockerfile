FROM golang:1.4.1-onbuild
ENTRYPOINT ["go-wrapper", "run"]
CMD ["-a", ":80"]
EXPOSE 80
