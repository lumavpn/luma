package rule

import (
	"github.com/lumavpn/luma/geodata/router"
)

type RuleGeoSite interface {
	GetDomainMatcher() router.DomainMatcher
}

type RuleGeoIP interface {
	GetIPMatcher() *router.GeoIPMatcher
}

type RuleGroup interface {
	GetRecodeSize() int
}
