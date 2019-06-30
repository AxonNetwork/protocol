package nodep2p

import (
	"context"
	"time"

	"github.com/Conscience/protocol/log"
)

func (h *Host) isAuthorised(repoID string, sig []byte) (bool, error) {
	addr, err := h.ethClient.AddrFromSignedHash([]byte(repoID), sig)
	if err != nil {
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	isAuth, err := h.ethClient.AddressHasPullAccess(ctx, addr, repoID)
	if err != nil {
		return false, err
	}

	if isAuth == false {
		log.Warnf("[p2p server] address 0x%0x does not have pull access", addr.Bytes())
	}

	return isAuth, nil
}
