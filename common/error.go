package common

import "errors"

var ErrRejectLoopback = errors.New("reject loopback connection")
