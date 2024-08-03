package box

import (
	"encoding/binary"
	"net"
)

type StatsMessage struct {
	Memory           int64
	Goroutines       int32
	BytesIn          int32
	BytesOut         int32
	TrafficAvailable bool
}

func (c *CommandClient) handleStatusConn(conn net.Conn) {
	for {
		var message StatsMessage
		err := binary.Read(conn, binary.BigEndian, &message)
		if err != nil {
			//c.handler.Disconnected(err.Error())
			return
		}
		c.handler.WriteStatus(&message)
	}
}
