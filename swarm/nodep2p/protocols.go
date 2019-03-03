package nodep2p

import (
	protocol "github.com/libp2p/go-libp2p-protocol"
)

const (
	MANIFEST_PROTO          protocol.ID = "/conscience/manifest/1.0.0"
	OBJECT_PROTO            protocol.ID = "/conscience/object/1.1.0"
	PACKFILE_PROTO          protocol.ID = "/conscience/packfile/1.0.0"
	CHUNK_PROTO             protocol.ID = "/conscience/chunk/1.1.0"
	REPLICATION_PROTO       protocol.ID = "/conscience/replication/1.1.0"
	BECOME_REPLICATOR_PROTO protocol.ID = "/conscience/become-replicator/1.1.0"
)

// @@TODO: make configurable
const OBJ_CHUNK_SIZE = 2 * 1024 * 1024 // 2mb
