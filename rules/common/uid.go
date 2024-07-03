package common

import (
	"fmt"
	"runtime"

	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/util"
)

type Uid struct {
	*Base
	uids    util.IntRanges[uint32]
	oUid    string
	adapter string
}

func NewUid(oUid, adapter string) (*Uid, error) {
	if !(runtime.GOOS == "linux" || runtime.GOOS == "android") {
		return nil, fmt.Errorf("uid rule not support this platform")
	}

	uidRange, err := util.NewUnsignedRanges[uint32](oUid)
	if err != nil {
		return nil, fmt.Errorf("%w, %w", errPayload, err)
	}

	if len(uidRange) == 0 {
		return nil, errPayload
	}
	return &Uid{
		Base:    &Base{},
		adapter: adapter,
		oUid:    oUid,
		uids:    uidRange,
	}, nil
}

func (u *Uid) RuleType() R.RuleType {
	return R.Uid
}

func (u *Uid) Match(metadata *M.Metadata) (bool, string) {
	if metadata.Uid != 0 {
		if u.uids.Check(metadata.Uid) {
			return true, u.adapter
		}
	}
	log.Warnf("[UID] could not get uid from %s", metadata.String())
	return false, ""
}

func (u *Uid) Adapter() string {
	return u.adapter
}

func (u *Uid) Payload() string {
	return u.oUid
}

func (u *Uid) ShouldFindProcess() bool {
	return true
}
