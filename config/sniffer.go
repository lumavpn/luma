package config

import (
	"fmt"
	"strings"

	"github.com/lumavpn/luma/component/trie"
	"github.com/lumavpn/luma/log"
	"github.com/lumavpn/luma/tunnel/sniffer"
	"github.com/lumavpn/luma/util"
)

type RawSniffingConfig struct {
	Ports        []string `yaml:"ports" json:"ports"`
	OverrideDest *bool    `yaml:"override-destination" json:"override-destination"`
}

type RawSniffer struct {
	Enable          bool                         `yaml:"enable" json:"enable"`
	OverrideDest    bool                         `yaml:"override-destination" json:"override-destination"`
	Sniffing        []string                     `yaml:"sniffing" json:"sniffing"`
	ForceDomain     []string                     `yaml:"force-domain" json:"force-domain"`
	SkipDomain      []string                     `yaml:"skip-domain" json:"skip-domain"`
	Ports           []string                     `yaml:"port-whitelist" json:"port-whitelist"`
	ForceDnsMapping bool                         `yaml:"force-dns-mapping" json:"force-dns-mapping"`
	ParsePureIp     bool                         `yaml:"parse-pure-ip" json:"parse-pure-ip"`
	Sniff           map[string]RawSniffingConfig `yaml:"sniff" json:"sniff"`
}

type Sniffer struct {
	Enable          bool
	Sniffers        map[sniffer.Type]sniffer.SnifferConfig
	ForceDomain     *trie.DomainSet
	SkipDomain      *trie.DomainSet
	ForceDnsMapping bool
	ParsePureIp     bool
}

func parseSniffer(snifferRaw RawSniffer) (*Sniffer, error) {
	s := &Sniffer{
		Enable:          snifferRaw.Enable,
		ForceDnsMapping: snifferRaw.ForceDnsMapping,
		ParsePureIp:     snifferRaw.ParsePureIp,
	}
	loadSniffer := make(map[sniffer.Type]sniffer.SnifferConfig)

	if len(snifferRaw.Sniff) != 0 {
		for sniffType, sniffConfig := range snifferRaw.Sniff {
			find := false
			ports, err := util.NewUnsignedRangesFromList[uint16](sniffConfig.Ports)
			if err != nil {
				return nil, err
			}
			overrideDest := snifferRaw.OverrideDest
			if sniffConfig.OverrideDest != nil {
				overrideDest = *sniffConfig.OverrideDest
			}
			for _, snifferType := range sniffer.List {
				if snifferType.String() == strings.ToUpper(sniffType) {
					find = true
					loadSniffer[snifferType] = sniffer.SnifferConfig{
						Ports:        ports,
						OverrideDest: overrideDest,
					}
				}
			}

			if !find {
				return nil, fmt.Errorf("not find the sniffer[%s]", sniffType)
			}
		}
	} else {
		if s.Enable {
			// Deprecated: Use Sniff instead
			log.Warn("Deprecated: Use Sniff instead")
		}
		globalPorts, err := util.NewUnsignedRangesFromList[uint16](snifferRaw.Ports)
		if err != nil {
			return nil, err
		}

		for _, snifferName := range snifferRaw.Sniffing {
			find := false
			for _, snifferType := range sniffer.List {
				if snifferType.String() == strings.ToUpper(snifferName) {
					find = true
					loadSniffer[snifferType] = sniffer.SnifferConfig{
						Ports:        globalPorts,
						OverrideDest: snifferRaw.OverrideDest,
					}
				}
			}

			if !find {
				return nil, fmt.Errorf("not find the sniffer[%s]", snifferName)
			}
		}
	}

	s.Sniffers = loadSniffer

	forceDomainTrie := trie.New[struct{}]()
	for _, domain := range snifferRaw.ForceDomain {
		err := forceDomainTrie.Insert(domain, struct{}{})
		if err != nil {
			return nil, fmt.Errorf("error domian[%s] in force-domain, error:%v", domain, err)
		}
	}
	s.ForceDomain = forceDomainTrie.NewDomainSet()

	skipDomainTrie := trie.New[struct{}]()
	for _, domain := range snifferRaw.SkipDomain {
		err := skipDomainTrie.Insert(domain, struct{}{})
		if err != nil {
			return nil, fmt.Errorf("error domian[%s] in force-domain, error:%v", domain, err)
		}
	}
	s.SkipDomain = skipDomainTrie.NewDomainSet()

	return s, nil
}
