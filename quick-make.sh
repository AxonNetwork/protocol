cd swarm/cmd
GO111MODULE=on go build -o main ./*.go
sudo mv main /usr/local/bin/conscience-node
cd ../../
