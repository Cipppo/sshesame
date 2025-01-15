FROM golang as build-env 

WORKDIR /go/src/sshesame

COPY ../*.go .
COPY ../go.mod .
COPY ../go.sum .
COPY ../sshesame.yaml .

RUN go build -o /go/bin/sshesame .

FROM ubuntu:latest
COPY --from=build-env /go/bin/sshesame /
COPY --from=build-env /go/src/sshesame/sshesame.yaml /config.yaml
RUN apt install libc6
EXPOSE 2022



ENTRYPOINT [ "/sshesame", "-config", "/config.yaml" ]
