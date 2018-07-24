# Node push/pull

## setup

first setup your testzone

1. `cd testzone`

2. `./setup.sh`

"original" is the existing repo. "testing" is the repo that you want to pull into

3. open two terminal windows

4. `./main 1337` in one

5. `./main 8080` in the other

6. add them as peers by doing `addrs` in one, copying the multiaddr with `127.0.0.1` in it and adding it to the other with `add-peer [multiaddr]`

7. add "original" to the node on `8080` and "testing" on the other with `add-repo [path/to/repo]`  

8. `cd` into "testing" and run `git pull`

It should pull all the git objects from original into it. Verify by checking `.git/objects`

Pull hook for conscience chunks is having issues right now. Will fix soon.

# p2p-dht

## building

forthcoming, it's kind of a pain.  for now, just use the `main` binary committed to this repo.

## usage

1. Open two terminal windows

2. In terminal 1, run `./main 1337` to start a node on port 1337.  This will output a key (beginning with `Qm`).

3. In terminal 2, run `./main 1338 1337/ipfs/Qm...` (with the key you got in step 2) to start a second node on port 1338 that's aware of the first node on 1337.

The second node will set a key in the DHT, and both nodes will reflect that.

# chunker

To try this out:

1. Compile the source in decode/encode/diff to binaries called 'conscience_decode', 'conscience_encode', and 'conscience_diff'

2. Put these binaries in your `$PATH`

3. Make an empty git repo somewhere

4. Open `.git/config` and add the following:

```
[filter "conscience"]
    clean = conscience_encode
    smudge = conscience_decode
[diff "conscience"]
	textconv = conscience_diff
```

5. Add a file called `.gitattributes` to the repo, with the following info:

```
*.gif        filter=conscience diff=conscience
```

6. Add `dabbing-boomer.gif` to the repo.

7. Commit `dabbing-boomer.gif`.  Then run `git show HEAD` to see what git actually committed (hint, it's not the binary data!)

8. Try deleting `dabbing-boomer.gif`.  Then run `git checkout -- dabbing-boomer.gif` to verify that Git can reconstruct it from the chunks.


