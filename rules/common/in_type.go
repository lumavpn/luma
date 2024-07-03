package common

import (
	"fmt"
	"strings"

	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	"github.com/lumavpn/luma/proxy/proto"
	R "github.com/lumavpn/luma/rule"
)

type InType struct {
	*Base
	types   []proto.Proto
	adapter string
	payload string
}

func (u *InType) Match(metadata *M.Metadata) (bool, string) {
	for _, tp := range u.types {
		if metadata.Type == tp {
			return true, u.adapter
		}
	}
	return false, ""
}

func (u *InType) RuleType() R.RuleType {
	return R.InType
}

func (u *InType) Adapter() string {
	return u.adapter
}

func (u *InType) Payload() string {
	return u.payload
}

func NewInType(iTypes, adapter string) (*InType, error) {
	types := strings.Split(iTypes, "/")
	if len(types) == 0 {
		return nil, fmt.Errorf("in type couldn't be empty")
	}

	tps, err := parseInTypes(types)
	if err != nil {
		return nil, err
	}

	return &InType{
		Base:    &Base{},
		types:   tps,
		adapter: adapter,
		payload: strings.ToUpper(iTypes),
	}, nil
}

func parseInTypes(tps []string) (res []proto.Proto, err error) {
	for _, tp := range tps {
		proxyProtocol, err := proto.EncodeProto(tp)
		if err != nil {
			log.Error(err)
			continue
		}
		res = append(res, proxyProtocol)
	}
	return
}
