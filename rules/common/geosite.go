package common

import (
	"fmt"

	"github.com/lumavpn/luma/geodata"
	_ "github.com/lumavpn/luma/geodata/memconservative"
	"github.com/lumavpn/luma/geodata/router"
	_ "github.com/lumavpn/luma/geodata/standard"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
)

type GEOSITE struct {
	*Base
	country    string
	adapter    string
	matcher    router.DomainMatcher
	recodeSize int
}

func (gs *GEOSITE) RuleType() R.RuleType {
	return R.GEOSITE
}

func (gs *GEOSITE) Match(metadata *M.Metadata) (bool, string) {
	domain := metadata.RuleHost()
	if len(domain) == 0 {
		return false, ""
	}
	return gs.matcher.ApplyDomain(domain), gs.adapter
}

func (gs *GEOSITE) Adapter() string {
	return gs.adapter
}

func (gs *GEOSITE) Payload() string {
	return gs.country
}

func (gs *GEOSITE) GetDomainMatcher() router.DomainMatcher {
	return gs.matcher
}

func (gs *GEOSITE) GetRecodeSize() int {
	return gs.recodeSize
}

func NewGEOSITE(country string, adapter string) (*GEOSITE, error) {
	if err := geodata.InitGeoSite(); err != nil {
		log.Errorf("can't initial GeoSite: %s", err)
		return nil, err
	}

	matcher, size, err := geodata.LoadGeoSiteMatcher(country)
	if err != nil {
		return nil, fmt.Errorf("load GeoSite data error, %s", err.Error())
	}

	log.Infof("Start initial GeoSite rule %s => %s, records: %d", country, adapter, size)

	geoSite := &GEOSITE{
		Base:       &Base{},
		country:    country,
		adapter:    adapter,
		matcher:    matcher,
		recodeSize: size,
	}

	return geoSite, nil
}

var _ R.Rule = (*GEOSITE)(nil)
