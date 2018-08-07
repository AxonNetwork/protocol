FROM golang:1.10.3-stretch

RUN go get -u github.com/whyrusleeping/gx
RUN go get -u github.com/whyrusleeping/gx-go

COPY . $GOPATH/src/github.com/Conscience/protocol
WORKDIR $GOPATH/src/github.com/Conscience/protocol
RUN gx install

WORKDIR $GOPATH/src/github.com/Conscience/protocol/swarm/cmd
RUN go get github.com/sirupsen/logrus
RUN go get github.com/BurntSushi/toml
RUN go get github.com/mitchellh/go-homedir
RUN go get github.com/pkg/errors
RUN go get github.com/ethereum/go-ethereum
RUN go get github.com/tyler-smith/go-bip39
RUN go get github.com/lunixbochs/struc
RUN go get github.com/btcsuite/btcd
RUN go get github.com/btcsuite/btcutil
RUN go get gopkg.in/src-d/go-git.v4

RUN go build -o /usr/local/bin/node main.go

CMD ["/usr/local/bin/node"]

