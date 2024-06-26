// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: proxies.proto

package proto

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

type Proto int32

const (
	Proto_PROTOCOL_UNSET Proto = 0
	Proto_HTTP           Proto = 1
	Proto_HTTPS          Proto = 2
	Proto_INNER          Proto = 3
	Proto_SOCKS4         Proto = 4
	Proto_SOCKS5         Proto = 5
	Proto_TUN            Proto = 6
	Proto_DIRECT         Proto = 7
)

// Enum value maps for Proto.
var (
	Proto_name = map[int32]string{
		0: "PROTOCOL_UNSET",
		1: "HTTP",
		2: "HTTPS",
		3: "INNER",
		4: "SOCKS4",
		5: "SOCKS5",
		6: "TUN",
		7: "DIRECT",
	}
	Proto_value = map[string]int32{
		"PROTOCOL_UNSET": 0,
		"HTTP":           1,
		"HTTPS":          2,
		"INNER":          3,
		"SOCKS4":         4,
		"SOCKS5":         5,
		"TUN":            6,
		"DIRECT":         7,
	}
)

func (x Proto) Enum() *Proto {
	p := new(Proto)
	*p = x
	return p
}

func (x Proto) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Proto) Descriptor() protoreflect.EnumDescriptor {
	return file_proxies_proto_enumTypes[0].Descriptor()
}

func (Proto) Type() protoreflect.EnumType {
	return &file_proxies_proto_enumTypes[0]
}

func (x Proto) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Proto.Descriptor instead.
func (Proto) EnumDescriptor() ([]byte, []int) {
	return file_proxies_proto_rawDescGZIP(), []int{0}
}

var File_proxies_proto protoreflect.FileDescriptor

var file_proxies_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x70, 0x72, 0x6f, 0x78, 0x69, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2a,
	0x68, 0x0a, 0x05, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x0a, 0x0e, 0x50, 0x52, 0x4f, 0x54,
	0x4f, 0x43, 0x4f, 0x4c, 0x5f, 0x55, 0x4e, 0x53, 0x45, 0x54, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04,
	0x48, 0x54, 0x54, 0x50, 0x10, 0x01, 0x12, 0x09, 0x0a, 0x05, 0x48, 0x54, 0x54, 0x50, 0x53, 0x10,
	0x02, 0x12, 0x09, 0x0a, 0x05, 0x49, 0x4e, 0x4e, 0x45, 0x52, 0x10, 0x03, 0x12, 0x0a, 0x0a, 0x06,
	0x53, 0x4f, 0x43, 0x4b, 0x53, 0x34, 0x10, 0x04, 0x12, 0x0a, 0x0a, 0x06, 0x53, 0x4f, 0x43, 0x4b,
	0x53, 0x35, 0x10, 0x05, 0x12, 0x07, 0x0a, 0x03, 0x54, 0x55, 0x4e, 0x10, 0x06, 0x12, 0x0a, 0x0a,
	0x06, 0x44, 0x49, 0x52, 0x45, 0x43, 0x54, 0x10, 0x07, 0x42, 0x25, 0x5a, 0x23, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6c, 0x75, 0x6d, 0x61, 0x76, 0x70, 0x6e, 0x2f,
	0x6c, 0x75, 0x6d, 0x61, 0x2f, 0x70, 0x72, 0x6f, 0x78, 0x79, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proxies_proto_rawDescOnce sync.Once
	file_proxies_proto_rawDescData = file_proxies_proto_rawDesc
)

func file_proxies_proto_rawDescGZIP() []byte {
	file_proxies_proto_rawDescOnce.Do(func() {
		file_proxies_proto_rawDescData = protoimpl.X.CompressGZIP(file_proxies_proto_rawDescData)
	})
	return file_proxies_proto_rawDescData
}

var file_proxies_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proxies_proto_goTypes = []any{
	(Proto)(0), // 0: Proto
}
var file_proxies_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proxies_proto_init() }
func file_proxies_proto_init() {
	if File_proxies_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proxies_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proxies_proto_goTypes,
		DependencyIndexes: file_proxies_proto_depIdxs,
		EnumInfos:         file_proxies_proto_enumTypes,
	}.Build()
	File_proxies_proto = out.File
	file_proxies_proto_rawDesc = nil
	file_proxies_proto_goTypes = nil
	file_proxies_proto_depIdxs = nil
}