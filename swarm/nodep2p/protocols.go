package nodep2p

import (
	protocol "github.com/libp2p/go-libp2p-protocol"
)

const (
	MANIFEST_PROTO    protocol.ID = "/axon/manifest/1.0.0"
	OBJECT_PROTO      protocol.ID = "/axon/object/1.1.0"
	PACKFILE_PROTO    protocol.ID = "/axon/packfile/1.0.0"
	CHUNK_PROTO       protocol.ID = "/axon/chunk/1.1.0"
	REPLICATION_PROTO protocol.ID = "/axon/replication/1.1.0"
)

// @@TODO: make configurable
const OBJ_CHUNK_SIZE = 2 * 1024 * 1024 // 2mb
