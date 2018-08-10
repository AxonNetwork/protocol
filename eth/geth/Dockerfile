FROM golang:1.10.3-stretch

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends git make gcc libc-dev ca-certificates && \
  git clone --depth 1 --branch release/1.8 https://github.com/ethereum/go-ethereum && \
  (cd go-ethereum && make geth) && \
  cp go-ethereum/build/bin/geth /geth && \
  apt-get remove -y git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /go-ethereum && mkdir /opt/ethdata

EXPOSE 8545
EXPOSE 30303

ADD ./ethdata /opt/ethdata
COPY efficient-mining.js /usr/local/efficient-mining.js

RUN /geth --networkid 23332 --datadir=/opt/ethdata --verbosity=4 init /opt/ethdata/genesis.json

CMD /geth --networkid 23332 \
          --datadir=/opt/ethdata \
          --targetgaslimit 7000000 \
          --gasprice "1000000000" \
          --verbosity=4 \
          --mine --minerthreads 1 \
          --nodiscover \
          --rpc --rpcport=8545 --rpcaddr=0.0.0.0 --rpccorsdomain "*" \
          --rpcvhosts=* \
          --etherbase abef44954d64269443125e86ce92b1d99c959017 \
          --unlock 0xabef44954d64269443125e86ce92b1d99c959017 \
          --password /opt/ethdata/node-password.txt \
          js /usr/local/efficient-mining.js
