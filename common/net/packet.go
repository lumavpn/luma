package net

import (
	"github.com/lumavpn/luma/common/net/deadline"
	"github.com/lumavpn/luma/common/net/packet"
)

type EnhancePacketConn = packet.EnhancePacketConn
type WaitReadFrom = packet.WaitReadFrom

var NewEnhancePacketConn = packet.NewEnhancePacketConn
var NewThreadSafePacketConn = packet.NewThreadSafePacketConn
var NewRefPacketConn = packet.NewRefPacketConn

var NewDeadlineNetPacketConn = deadline.NewNetPacketConn
var NewDeadlineEnhancePacketConn = deadline.NewEnhancePacketConn
var NewDeadlineSingPacketConn = deadline.NewSingPacketConn
var NewDeadlineEnhanceSingPacketConn = deadline.NewEnhanceSingPacketConn
