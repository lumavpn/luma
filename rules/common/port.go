package common

import (
	"fmt"

	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
	"github.com/lumavpn/luma/util"
)

type Port struct {
	*Base
	adapter    string
	port       string
	ruleType   R.RuleType
	portRanges util.IntRanges[uint16]
}

func (p *Port) RuleType() R.RuleType {
	return p.ruleType
}

func (p *Port) Match(metadata *M.Metadata) (bool, string) {
	targetPort := metadata.DstPort
	switch p.ruleType {
	case R.InPort:
		targetPort = metadata.InPort
	case R.SrcPort:
		targetPort = metadata.SrcPort
	}
	return p.portRanges.Check(targetPort), p.adapter
}

func (p *Port) Adapter() string {
	return p.adapter
}

func (p *Port) Payload() string {
	return p.port
}

func NewPort(port string, adapter string, ruleType R.RuleType) (*Port, error) {
	portRanges, err := util.NewUnsignedRanges[uint16](port)
	if err != nil {
		return nil, fmt.Errorf("%w, %w", errPayload, err)
	}

	if len(portRanges) == 0 {
		return nil, errPayload
	}

	return &Port{
		Base:       &Base{},
		adapter:    adapter,
		port:       port,
		ruleType:   ruleType,
		portRanges: portRanges,
	}, nil
}

var _ R.Rule = (*Port)(nil)
