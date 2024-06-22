package tunnel

import (
	"github.com/lumavpn/luma/adapter"
	"github.com/lumavpn/luma/log"
)

func (t *tunnel) handleUDPConn(packet adapter.PacketAdapter) {
	if !t.isHandle(packet.Metadata().Type) {
		packet.Drop()
		return
	}

	metadata := packet.Metadata()
	if !metadata.Valid() {
		packet.Drop()
		log.Debugf("[Metadata] not valid: %#v", metadata)
		return
	}

	if err := preHandleMetadata(metadata); err != nil {
		packet.Drop()
		log.Debugf("[Metadata PreHandle] error: %s", err)
		return
	}
}
