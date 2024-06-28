package adapter

import (
	"github.com/gofrs/uuid/v5"
	"github.com/lumavpn/luma/metadata"
)

// ConnContext is the default interface to adapt connections
type ConnContext interface {
	ID() uuid.UUID
	Metadata() *metadata.Metadata
}
