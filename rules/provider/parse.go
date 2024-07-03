package provider

import (
	"errors"
	"fmt"
	"time"

	C "github.com/lumavpn/luma/common"
	CP "github.com/lumavpn/luma/common"
	"github.com/lumavpn/luma/common/structure"
	"github.com/lumavpn/luma/component/resource"
	"github.com/lumavpn/luma/features"
	P "github.com/lumavpn/luma/proxy/provider"
	R "github.com/lumavpn/luma/rule"
)

var (
	errSubPath = errors.New("path is not subpath of home directory")
)

type ruleProviderSchema struct {
	Type     string `provider:"type"`
	Behavior string `provider:"behavior"`
	Path     string `provider:"path,omitempty"`
	URL      string `provider:"url,omitempty"`
	Proxy    string `provider:"proxy,omitempty"`
	Format   string `provider:"format,omitempty"`
	Interval int    `provider:"interval,omitempty"`
}

func ParseRuleProvider(name string, mapping map[string]interface{}, parse func(tp, payload, target string, params []string, subRules map[string][]R.Rule) (parsed R.Rule, parseErr error)) (P.RuleProvider, error) {
	schema := &ruleProviderSchema{}
	decoder := structure.NewDecoder(structure.Option{TagName: "provider", WeaklyTypedInput: true})
	if err := decoder.Decode(mapping, schema); err != nil {
		return nil, err
	}
	var behavior P.RuleBehavior

	switch schema.Behavior {
	case "domain":
		behavior = P.Domain
	case "ipcidr":
		behavior = P.IPCIDR
	case "classical":
		behavior = P.Classical
	default:
		return nil, fmt.Errorf("unsupported behavior type: %s", schema.Behavior)
	}

	var format P.RuleFormat

	switch schema.Format {
	case "", "yaml":
		format = P.YamlRule
	case "text":
		format = P.TextRule
	default:
		return nil, fmt.Errorf("unsupported format type: %s", schema.Format)
	}

	var vehicle C.Vehicle
	switch schema.Type {
	case "file":
		path := CP.Path.Resolve(schema.Path)
		vehicle = resource.NewFileVehicle(path)
	case "http":
		path := CP.Path.GetPathByHash("rules", schema.URL)
		if schema.Path != "" {
			path = CP.Path.Resolve(schema.Path)
			if !features.CMFA && !CP.Path.IsSafePath(path) {
				return nil, fmt.Errorf("%w: %s", errSubPath, path)
			}
		}
		vehicle = resource.NewHTTPVehicle(schema.URL, path, schema.Proxy, nil)
	default:
		return nil, fmt.Errorf("unsupported vehicle type: %s", schema.Type)
	}

	return NewRuleSetProvider(name, behavior, format, time.Duration(uint(schema.Interval))*time.Second, vehicle, parse), nil
}
