FROM golang:1.4.1-onbuild
CMD ["go-wrapper", "run", "-a", ":80"]
EXPOSE 80
