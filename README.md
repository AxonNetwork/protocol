# p2p-dht

## building

forthcoming, it's kind of a pain.  for now, just use the `main` binary committed to this repo.

## usage

1. Open two terminal windows

2. In terminal 1, run `./main 1337` to start a node on port 1337.  This will output a key (beginning with `Qm`).

3. In terminal 2, run `./main 1338 1337/ipfs/Qm...` (with the key you got in step 2) to start a second node on port 1338 that's aware of the first node on 1337.

The second node will set a key in the DHT, and both nodes will reflect that.
