package common

import (
	"fmt"
	"strings"

	C "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/component/mmdb"
	"github.com/lumavpn/luma/component/resolver"
	"github.com/lumavpn/luma/geodata"
	"github.com/lumavpn/luma/geodata/router"
	"github.com/lumavpn/luma/log"
	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
)

type GEOIP struct {
	*Base
	country      string
	adapter      string
	noResolveIP  bool
	isSourceIP   bool
	geoIPMatcher *router.GeoIPMatcher
	recodeSize   int
}

var _ R.Rule = (*GEOIP)(nil)

func (g *GEOIP) RuleType() R.RuleType {
	if g.isSourceIP {
		return R.SrcGEOIP
	}
	return R.GEOIP
}

func (g *GEOIP) Match(metadata *M.Metadata) (bool, string) {
	ip := metadata.DstIP
	if g.isSourceIP {
		ip = metadata.SrcIP
	}
	if !ip.IsValid() {
		return false, ""
	}

	if g.country == "lan" {
		return ip.IsPrivate() ||
			ip.IsUnspecified() ||
			ip.IsLoopback() ||
			ip.IsMulticast() ||
			ip.IsLinkLocalUnicast() ||
			resolver.IsFakeBroadcastIP(ip), g.adapter
	}

	for _, code := range metadata.DstGeoIP {
		if g.country == code {
			return true, g.adapter
		}
	}

	if !C.GeodataMode {
		if g.isSourceIP {
			codes := mmdb.IPInstance().LookupCode(ip.AsSlice())
			for _, code := range codes {
				if g.country == code {
					return true, g.adapter
				}
			}
			return false, g.adapter
		}

		if metadata.DstGeoIP != nil {
			return false, g.adapter
		}
		metadata.DstGeoIP = mmdb.IPInstance().LookupCode(ip.AsSlice())
		for _, code := range metadata.DstGeoIP {
			if g.country == code {
				return true, g.adapter
			}
		}
		return false, g.adapter
	}

	match := g.geoIPMatcher.Match(ip)
	if match && !g.isSourceIP {
		metadata.DstGeoIP = append(metadata.DstGeoIP, g.country)
	}
	return match, g.adapter
}

func (g *GEOIP) Adapter() string {
	return g.adapter
}

func (g *GEOIP) Payload() string {
	return g.country
}

func (g *GEOIP) ShouldResolveIP() bool {
	return !g.noResolveIP
}

func (g *GEOIP) GetCountry() string {
	return g.country
}

func (g *GEOIP) GetIPMatcher() *router.GeoIPMatcher {
	return g.geoIPMatcher
}

func (g *GEOIP) GetRecodeSize() int {
	return g.recodeSize
}

func NewGEOIP(country string, adapter string, isSrc, noResolveIP bool) (*GEOIP, error) {
	if err := geodata.InitGeoIP(); err != nil {
		log.Errorf("can't initial GeoIP: %s", err)
		return nil, err
	}
	country = strings.ToLower(country)

	if !C.GeodataMode || country == "lan" {
		geoip := &GEOIP{
			Base:        &Base{},
			country:     country,
			adapter:     adapter,
			noResolveIP: noResolveIP,
			isSourceIP:  isSrc,
		}
		return geoip, nil
	}

	geoIPMatcher, size, err := geodata.LoadGeoIPMatcher(country)
	if err != nil {
		return nil, fmt.Errorf("[GeoIP] %w", err)
	}

	log.Infof("Start initial GeoIP rule %s => %s, records: %d", country, adapter, size)
	geoip := &GEOIP{
		Base:         &Base{},
		country:      country,
		adapter:      adapter,
		noResolveIP:  noResolveIP,
		isSourceIP:   isSrc,
		geoIPMatcher: geoIPMatcher,
		recodeSize:   size,
	}
	return geoip, nil
}
