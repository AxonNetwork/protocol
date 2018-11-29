# benchmarks

## create manifest

Benchmarks the traversal of the entire object graph from HEAD backwards.

```sh
$ go run create-manifest.go <repo path 1> [optional repo path 2]
```

To unpack a repo's packfiles (for comparison testing):

```sh
$ mv ./.git/objects/pack/*.pack ../
$ rm ./.git/objects/pack/*.idx
$ cat ../<whatever>.pack | git unpack-objects
```

## p2p raw transfer

Benchmarks the transfer of a large chunk of raw data with no processing at either end of the connection.

Run the server: 

```sh
$ go run p2p-raw-transfer.go
```

Run the client:

```sh
$ go run p2p-raw-transfer.go <server multiaddress>
```

