package box

import (
	"encoding/binary"
	"net"
	"path/filepath"
	"time"

	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
)

type CommandClient struct {
	handler CommandClientHandler
	conn    net.Conn
	options CommandClientOptions
}

type CommandClientOptions struct {
	Command        int32
	StatusInterval int64
}

type CommandClientHandler interface {
	WriteStatus(message *StatsMessage)
}

func NewCommandClient(handler CommandClientHandler, options *CommandClientOptions) *CommandClient {
	return &CommandClient{
		handler: handler,
		options: common.PtrValueOrDefault(options),
	}
}

func (c *CommandClient) directConnect() (net.Conn, error) {
	if !sTVOS {
		return net.DialUnix("unix", nil, &net.UnixAddr{
			Name: filepath.Join(sBasePath, "command.sock"),
			Net:  "unix",
		})
	} else {
		return net.Dial("tcp", "127.0.0.1:8964")
	}
}

func (c *CommandClient) directConnectWithRetry() (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	for i := 0; i < 10; i++ {
		conn, err = c.directConnect()
		if err == nil {
			return conn, nil
		}
		time.Sleep(time.Duration(100+i*50) * time.Millisecond)
	}
	return nil, err
}

func (c *CommandClient) Connect() error {
	common.Close(c.conn)
	conn, err := c.directConnectWithRetry()
	if err != nil {
		return err
	}
	c.conn = conn
	err = binary.Write(conn, binary.BigEndian, uint8(c.options.Command))
	if err != nil {
		return err
	}
	switch c.options.Command {
	case CommandStatus:
		err = binary.Write(conn, binary.BigEndian, c.options.StatusInterval)
		if err != nil {
			return E.Cause(err, "write interval")
		}
		//c.handler.Connected()
		go c.handleStatusConn(conn)
	}
	return nil
}

func (c *CommandClient) Disconnect() error {
	return nil
	//return common.Close(c.conn)
}
