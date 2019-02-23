cd swarm/cmd
GO111MODULE=on go build -o main ./*.go
sudo mv main /usr/local/bin/conscience-node
cd ../../

cd remote-helper
GO111MODULE=on go build -o main ./*.go
mv main /usr/local/bin/git-remote-conscience
cd ../

cd filters/encode
GO111MODULE=on go build -o main ./*.go
sudo mv main /usr/local/bin/conscience_encode
cd ../../

cd filters/decode
GO111MODULE=on go build -o main ./*.go
sudo mv main /usr/local/bin/conscience_decode
cd ../../
