FROM golang:1.4.2

RUN mkdir -p /go/src/github.com/samertm/githubstreaks
WORKDIR /go/src/github.com/samertm/githubstreaks

COPY . /go/src/github.com/samertm/githubstreaks

RUN ln -sf conf.prod.toml conf.toml

RUN go get -v github.com/samertm/githubstreaks

CMD ["githubstreaks"]

EXPOSE 8000
