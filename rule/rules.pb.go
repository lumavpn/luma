// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: rules.proto

package rule

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RuleType int32

const (
	RuleType_Unset         RuleType = 0
	RuleType_DOMAIN        RuleType = 1
	RuleType_DOMAINSUFFIX  RuleType = 2
	RuleType_DOMAINKEYWORD RuleType = 3
	RuleType_DOMAINREGEX   RuleType = 4
	RuleType_GEOSITE       RuleType = 5
	RuleType_GEOIP         RuleType = 6
	RuleType_SRCPORT       RuleType = 7
	RuleType_DSTPORT       RuleType = 8
	RuleType_NETWORK       RuleType = 9
)

// Enum value maps for RuleType.
var (
	RuleType_name = map[int32]string{
		0: "Unset",
		1: "DOMAIN",
		2: "DOMAINSUFFIX",
		3: "DOMAINKEYWORD",
		4: "DOMAINREGEX",
		5: "GEOSITE",
		6: "GEOIP",
		7: "SRCPORT",
		8: "DSTPORT",
		9: "NETWORK",
	}
	RuleType_value = map[string]int32{
		"Unset":         0,
		"DOMAIN":        1,
		"DOMAINSUFFIX":  2,
		"DOMAINKEYWORD": 3,
		"DOMAINREGEX":   4,
		"GEOSITE":       5,
		"GEOIP":         6,
		"SRCPORT":       7,
		"DSTPORT":       8,
		"NETWORK":       9,
	}
)

func (x RuleType) Enum() *RuleType {
	p := new(RuleType)
	*p = x
	return p
}

func (x RuleType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (RuleType) Descriptor() protoreflect.EnumDescriptor {
	return file_rules_proto_enumTypes[0].Descriptor()
}

func (RuleType) Type() protoreflect.EnumType {
	return &file_rules_proto_enumTypes[0]
}

func (x RuleType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use RuleType.Descriptor instead.
func (RuleType) EnumDescriptor() ([]byte, []int) {
	return file_rules_proto_rawDescGZIP(), []int{0}
}

var File_rules_proto protoreflect.FileDescriptor

var file_rules_proto_rawDesc = []byte{
	0x0a, 0x0b, 0x72, 0x75, 0x6c, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2a, 0x96, 0x01,
	0x0a, 0x08, 0x52, 0x75, 0x6c, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x09, 0x0a, 0x05, 0x55, 0x6e,
	0x73, 0x65, 0x74, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x4f, 0x4d, 0x41, 0x49, 0x4e, 0x10,
	0x01, 0x12, 0x10, 0x0a, 0x0c, 0x44, 0x4f, 0x4d, 0x41, 0x49, 0x4e, 0x53, 0x55, 0x46, 0x46, 0x49,
	0x58, 0x10, 0x02, 0x12, 0x11, 0x0a, 0x0d, 0x44, 0x4f, 0x4d, 0x41, 0x49, 0x4e, 0x4b, 0x45, 0x59,
	0x57, 0x4f, 0x52, 0x44, 0x10, 0x03, 0x12, 0x0f, 0x0a, 0x0b, 0x44, 0x4f, 0x4d, 0x41, 0x49, 0x4e,
	0x52, 0x45, 0x47, 0x45, 0x58, 0x10, 0x04, 0x12, 0x0b, 0x0a, 0x07, 0x47, 0x45, 0x4f, 0x53, 0x49,
	0x54, 0x45, 0x10, 0x05, 0x12, 0x09, 0x0a, 0x05, 0x47, 0x45, 0x4f, 0x49, 0x50, 0x10, 0x06, 0x12,
	0x0b, 0x0a, 0x07, 0x53, 0x52, 0x43, 0x50, 0x4f, 0x52, 0x54, 0x10, 0x07, 0x12, 0x0b, 0x0a, 0x07,
	0x44, 0x53, 0x54, 0x50, 0x4f, 0x52, 0x54, 0x10, 0x08, 0x12, 0x0b, 0x0a, 0x07, 0x4e, 0x45, 0x54,
	0x57, 0x4f, 0x52, 0x4b, 0x10, 0x09, 0x42, 0x08, 0x5a, 0x06, 0x2e, 0x2f, 0x72, 0x75, 0x6c, 0x65,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_rules_proto_rawDescOnce sync.Once
	file_rules_proto_rawDescData = file_rules_proto_rawDesc
)

func file_rules_proto_rawDescGZIP() []byte {
	file_rules_proto_rawDescOnce.Do(func() {
		file_rules_proto_rawDescData = protoimpl.X.CompressGZIP(file_rules_proto_rawDescData)
	})
	return file_rules_proto_rawDescData
}

var file_rules_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_rules_proto_goTypes = []any{
	(RuleType)(0), // 0: RuleType
}
var file_rules_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_rules_proto_init() }
func file_rules_proto_init() {
	if File_rules_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_rules_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_rules_proto_goTypes,
		DependencyIndexes: file_rules_proto_depIdxs,
		EnumInfos:         file_rules_proto_enumTypes,
	}.Build()
	File_rules_proto = out.File
	file_rules_proto_rawDesc = nil
	file_rules_proto_goTypes = nil
	file_rules_proto_depIdxs = nil
}