package nodep2p

const (
	// OBJECT_PROTO    = "/conscience/object/1.0.0"
	MANIFEST_PROTO          = "/conscience/manifest/1.0.0"
	OBJECT_PROTO            = "/conscience/object/1.1.0"
	PACKFILE_PROTO          = "/conscience/packfile/1.0.0"
	REPLICATION_PROTO       = "/conscience/replication/1.1.0"
	BECOME_REPLICATOR_PROTO = "/conscience/become-replicator/1.1.0"
)

// @@TODO: make configurable
const OBJ_CHUNK_SIZE = 2 * 1024 * 1024 // 2mb
