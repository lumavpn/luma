package common

import (
	"fmt"

	M "github.com/lumavpn/luma/metadata"
	C "github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/util"
)

type DSCP struct {
	*Base
	ranges  util.IntRanges[uint8]
	payload string
	adapter string
}

func (d *DSCP) RuleType() C.RuleType {
	return C.DSCP
}

func (d *DSCP) Match(metadata *M.Metadata) (bool, string) {
	return d.ranges.Check(metadata.DSCP), d.adapter
}

func (d *DSCP) Adapter() string {
	return d.adapter
}

func (d *DSCP) Payload() string {
	return d.payload
}

func NewDSCP(dscp string, adapter string) (*DSCP, error) {
	ranges, err := util.NewUnsignedRanges[uint8](dscp)
	if err != nil {
		return nil, fmt.Errorf("parse DSCP rule fail: %w", err)
	}
	for _, r := range ranges {
		if r.End() > 63 {
			return nil, fmt.Errorf("DSCP couldn't be negative or exceed 63")
		}
	}
	return &DSCP{
		Base:    &Base{},
		payload: dscp,
		ranges:  ranges,
		adapter: adapter,
	}, nil
}
