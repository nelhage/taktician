// Code generated by protoc-gen-go. DO NOT EDIT.
// source: tak/proto/taktician.proto

package pb

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type AnalyzeRequest struct {
	Position             string   `protobuf:"bytes,1,opt,name=position" json:"position,omitempty"`
	Depth                int32    `protobuf:"varint,2,opt,name=depth" json:"depth,omitempty"`
	Precise              bool     `protobuf:"varint,3,opt,name=precise" json:"precise,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AnalyzeRequest) Reset()         { *m = AnalyzeRequest{} }
func (m *AnalyzeRequest) String() string { return proto.CompactTextString(m) }
func (*AnalyzeRequest) ProtoMessage()    {}
func (*AnalyzeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_taktician_7c7e336b14965d18, []int{0}
}
func (m *AnalyzeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AnalyzeRequest.Unmarshal(m, b)
}
func (m *AnalyzeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AnalyzeRequest.Marshal(b, m, deterministic)
}
func (dst *AnalyzeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AnalyzeRequest.Merge(dst, src)
}
func (m *AnalyzeRequest) XXX_Size() int {
	return xxx_messageInfo_AnalyzeRequest.Size(m)
}
func (m *AnalyzeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_AnalyzeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_AnalyzeRequest proto.InternalMessageInfo

func (m *AnalyzeRequest) GetPosition() string {
	if m != nil {
		return m.Position
	}
	return ""
}

func (m *AnalyzeRequest) GetDepth() int32 {
	if m != nil {
		return m.Depth
	}
	return 0
}

func (m *AnalyzeRequest) GetPrecise() bool {
	if m != nil {
		return m.Precise
	}
	return false
}

type AnalyzeResponse struct {
	Pv                   []string `protobuf:"bytes,1,rep,name=pv" json:"pv,omitempty"`
	Value                int64    `protobuf:"varint,2,opt,name=value" json:"value,omitempty"`
	Depth                int32    `protobuf:"varint,3,opt,name=depth" json:"depth,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AnalyzeResponse) Reset()         { *m = AnalyzeResponse{} }
func (m *AnalyzeResponse) String() string { return proto.CompactTextString(m) }
func (*AnalyzeResponse) ProtoMessage()    {}
func (*AnalyzeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_taktician_7c7e336b14965d18, []int{1}
}
func (m *AnalyzeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AnalyzeResponse.Unmarshal(m, b)
}
func (m *AnalyzeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AnalyzeResponse.Marshal(b, m, deterministic)
}
func (dst *AnalyzeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AnalyzeResponse.Merge(dst, src)
}
func (m *AnalyzeResponse) XXX_Size() int {
	return xxx_messageInfo_AnalyzeResponse.Size(m)
}
func (m *AnalyzeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_AnalyzeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_AnalyzeResponse proto.InternalMessageInfo

func (m *AnalyzeResponse) GetPv() []string {
	if m != nil {
		return m.Pv
	}
	return nil
}

func (m *AnalyzeResponse) GetValue() int64 {
	if m != nil {
		return m.Value
	}
	return 0
}

func (m *AnalyzeResponse) GetDepth() int32 {
	if m != nil {
		return m.Depth
	}
	return 0
}

type CanonicalizeRequest struct {
	Size                 int32    `protobuf:"varint,1,opt,name=size" json:"size,omitempty"`
	Moves                []string `protobuf:"bytes,2,rep,name=moves" json:"moves,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CanonicalizeRequest) Reset()         { *m = CanonicalizeRequest{} }
func (m *CanonicalizeRequest) String() string { return proto.CompactTextString(m) }
func (*CanonicalizeRequest) ProtoMessage()    {}
func (*CanonicalizeRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_taktician_7c7e336b14965d18, []int{2}
}
func (m *CanonicalizeRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CanonicalizeRequest.Unmarshal(m, b)
}
func (m *CanonicalizeRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CanonicalizeRequest.Marshal(b, m, deterministic)
}
func (dst *CanonicalizeRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CanonicalizeRequest.Merge(dst, src)
}
func (m *CanonicalizeRequest) XXX_Size() int {
	return xxx_messageInfo_CanonicalizeRequest.Size(m)
}
func (m *CanonicalizeRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_CanonicalizeRequest.DiscardUnknown(m)
}

var xxx_messageInfo_CanonicalizeRequest proto.InternalMessageInfo

func (m *CanonicalizeRequest) GetSize() int32 {
	if m != nil {
		return m.Size
	}
	return 0
}

func (m *CanonicalizeRequest) GetMoves() []string {
	if m != nil {
		return m.Moves
	}
	return nil
}

type CanonicalizeResponse struct {
	Moves                []string `protobuf:"bytes,1,rep,name=moves" json:"moves,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CanonicalizeResponse) Reset()         { *m = CanonicalizeResponse{} }
func (m *CanonicalizeResponse) String() string { return proto.CompactTextString(m) }
func (*CanonicalizeResponse) ProtoMessage()    {}
func (*CanonicalizeResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_taktician_7c7e336b14965d18, []int{3}
}
func (m *CanonicalizeResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CanonicalizeResponse.Unmarshal(m, b)
}
func (m *CanonicalizeResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CanonicalizeResponse.Marshal(b, m, deterministic)
}
func (dst *CanonicalizeResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CanonicalizeResponse.Merge(dst, src)
}
func (m *CanonicalizeResponse) XXX_Size() int {
	return xxx_messageInfo_CanonicalizeResponse.Size(m)
}
func (m *CanonicalizeResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_CanonicalizeResponse.DiscardUnknown(m)
}

var xxx_messageInfo_CanonicalizeResponse proto.InternalMessageInfo

func (m *CanonicalizeResponse) GetMoves() []string {
	if m != nil {
		return m.Moves
	}
	return nil
}

type IsPositionInTakRequest struct {
	Position             string   `protobuf:"bytes,1,opt,name=position" json:"position,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IsPositionInTakRequest) Reset()         { *m = IsPositionInTakRequest{} }
func (m *IsPositionInTakRequest) String() string { return proto.CompactTextString(m) }
func (*IsPositionInTakRequest) ProtoMessage()    {}
func (*IsPositionInTakRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_taktician_7c7e336b14965d18, []int{4}
}
func (m *IsPositionInTakRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IsPositionInTakRequest.Unmarshal(m, b)
}
func (m *IsPositionInTakRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IsPositionInTakRequest.Marshal(b, m, deterministic)
}
func (dst *IsPositionInTakRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IsPositionInTakRequest.Merge(dst, src)
}
func (m *IsPositionInTakRequest) XXX_Size() int {
	return xxx_messageInfo_IsPositionInTakRequest.Size(m)
}
func (m *IsPositionInTakRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_IsPositionInTakRequest.DiscardUnknown(m)
}

var xxx_messageInfo_IsPositionInTakRequest proto.InternalMessageInfo

func (m *IsPositionInTakRequest) GetPosition() string {
	if m != nil {
		return m.Position
	}
	return ""
}

type IsPositionInTakResponse struct {
	InTak                bool     `protobuf:"varint,1,opt,name=inTak" json:"inTak,omitempty"`
	TakMove              string   `protobuf:"bytes,2,opt,name=takMove" json:"takMove,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *IsPositionInTakResponse) Reset()         { *m = IsPositionInTakResponse{} }
func (m *IsPositionInTakResponse) String() string { return proto.CompactTextString(m) }
func (*IsPositionInTakResponse) ProtoMessage()    {}
func (*IsPositionInTakResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_taktician_7c7e336b14965d18, []int{5}
}
func (m *IsPositionInTakResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IsPositionInTakResponse.Unmarshal(m, b)
}
func (m *IsPositionInTakResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IsPositionInTakResponse.Marshal(b, m, deterministic)
}
func (dst *IsPositionInTakResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IsPositionInTakResponse.Merge(dst, src)
}
func (m *IsPositionInTakResponse) XXX_Size() int {
	return xxx_messageInfo_IsPositionInTakResponse.Size(m)
}
func (m *IsPositionInTakResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_IsPositionInTakResponse.DiscardUnknown(m)
}

var xxx_messageInfo_IsPositionInTakResponse proto.InternalMessageInfo

func (m *IsPositionInTakResponse) GetInTak() bool {
	if m != nil {
		return m.InTak
	}
	return false
}

func (m *IsPositionInTakResponse) GetTakMove() string {
	if m != nil {
		return m.TakMove
	}
	return ""
}

func init() {
	proto.RegisterType((*AnalyzeRequest)(nil), "tak.proto.AnalyzeRequest")
	proto.RegisterType((*AnalyzeResponse)(nil), "tak.proto.AnalyzeResponse")
	proto.RegisterType((*CanonicalizeRequest)(nil), "tak.proto.CanonicalizeRequest")
	proto.RegisterType((*CanonicalizeResponse)(nil), "tak.proto.CanonicalizeResponse")
	proto.RegisterType((*IsPositionInTakRequest)(nil), "tak.proto.IsPositionInTakRequest")
	proto.RegisterType((*IsPositionInTakResponse)(nil), "tak.proto.IsPositionInTakResponse")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Taktician service

type TakticianClient interface {
	Analyze(ctx context.Context, in *AnalyzeRequest, opts ...grpc.CallOption) (*AnalyzeResponse, error)
	Canonicalize(ctx context.Context, in *CanonicalizeRequest, opts ...grpc.CallOption) (*CanonicalizeResponse, error)
	IsPositionInTak(ctx context.Context, in *IsPositionInTakRequest, opts ...grpc.CallOption) (*IsPositionInTakResponse, error)
}

type takticianClient struct {
	cc *grpc.ClientConn
}

func NewTakticianClient(cc *grpc.ClientConn) TakticianClient {
	return &takticianClient{cc}
}

func (c *takticianClient) Analyze(ctx context.Context, in *AnalyzeRequest, opts ...grpc.CallOption) (*AnalyzeResponse, error) {
	out := new(AnalyzeResponse)
	err := grpc.Invoke(ctx, "/tak.proto.Taktician/Analyze", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *takticianClient) Canonicalize(ctx context.Context, in *CanonicalizeRequest, opts ...grpc.CallOption) (*CanonicalizeResponse, error) {
	out := new(CanonicalizeResponse)
	err := grpc.Invoke(ctx, "/tak.proto.Taktician/Canonicalize", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *takticianClient) IsPositionInTak(ctx context.Context, in *IsPositionInTakRequest, opts ...grpc.CallOption) (*IsPositionInTakResponse, error) {
	out := new(IsPositionInTakResponse)
	err := grpc.Invoke(ctx, "/tak.proto.Taktician/IsPositionInTak", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Taktician service

type TakticianServer interface {
	Analyze(context.Context, *AnalyzeRequest) (*AnalyzeResponse, error)
	Canonicalize(context.Context, *CanonicalizeRequest) (*CanonicalizeResponse, error)
	IsPositionInTak(context.Context, *IsPositionInTakRequest) (*IsPositionInTakResponse, error)
}

func RegisterTakticianServer(s *grpc.Server, srv TakticianServer) {
	s.RegisterService(&_Taktician_serviceDesc, srv)
}

func _Taktician_Analyze_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AnalyzeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TakticianServer).Analyze(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/tak.proto.Taktician/Analyze",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TakticianServer).Analyze(ctx, req.(*AnalyzeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Taktician_Canonicalize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CanonicalizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TakticianServer).Canonicalize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/tak.proto.Taktician/Canonicalize",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TakticianServer).Canonicalize(ctx, req.(*CanonicalizeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Taktician_IsPositionInTak_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(IsPositionInTakRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TakticianServer).IsPositionInTak(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/tak.proto.Taktician/IsPositionInTak",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TakticianServer).IsPositionInTak(ctx, req.(*IsPositionInTakRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Taktician_serviceDesc = grpc.ServiceDesc{
	ServiceName: "tak.proto.Taktician",
	HandlerType: (*TakticianServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Analyze",
			Handler:    _Taktician_Analyze_Handler,
		},
		{
			MethodName: "Canonicalize",
			Handler:    _Taktician_Canonicalize_Handler,
		},
		{
			MethodName: "IsPositionInTak",
			Handler:    _Taktician_IsPositionInTak_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "tak/proto/taktician.proto",
}

func init() {
	proto.RegisterFile("tak/proto/taktician.proto", fileDescriptor_taktician_7c7e336b14965d18)
}

var fileDescriptor_taktician_7c7e336b14965d18 = []byte{
	// 343 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x51, 0xc1, 0x4f, 0xfa, 0x30,
	0x14, 0xfe, 0x6d, 0x83, 0x1f, 0xec, 0xc5, 0x40, 0x52, 0x89, 0x8e, 0x1d, 0x74, 0xf6, 0xb4, 0x83,
	0x81, 0x44, 0xbd, 0x1b, 0xf1, 0xc4, 0x81, 0x44, 0x1b, 0x4e, 0xc4, 0x4b, 0xc1, 0x26, 0x36, 0xc3,
	0xb6, 0xd2, 0xb2, 0x44, 0xfe, 0x78, 0x63, 0xd6, 0xb2, 0x31, 0x14, 0x8d, 0xb7, 0x7e, 0x7d, 0xdf,
	0xfb, 0xbe, 0xf7, 0xbe, 0x07, 0x7d, 0x43, 0xb3, 0xa1, 0x5a, 0x49, 0x23, 0x87, 0x86, 0x66, 0x86,
	0x2f, 0x38, 0x15, 0x03, 0x8b, 0x51, 0x68, 0x68, 0xe6, 0x9e, 0xf8, 0x09, 0x3a, 0x77, 0x82, 0x2e,
	0xdf, 0x37, 0x8c, 0xb0, 0xb7, 0x35, 0xd3, 0x06, 0xc5, 0xd0, 0x56, 0x52, 0x73, 0xc3, 0xa5, 0x88,
	0xbc, 0xc4, 0x4b, 0x43, 0x52, 0x61, 0xd4, 0x83, 0xe6, 0x33, 0x53, 0xe6, 0x25, 0xf2, 0x13, 0x2f,
	0x6d, 0x12, 0x07, 0x50, 0x04, 0x2d, 0xb5, 0x62, 0x0b, 0xae, 0x59, 0x14, 0x24, 0x5e, 0xda, 0x26,
	0x25, 0xc4, 0x13, 0xe8, 0x56, 0xea, 0x5a, 0x49, 0xa1, 0x19, 0xea, 0x80, 0xaf, 0xf2, 0xc8, 0x4b,
	0x82, 0x34, 0x24, 0xbe, 0xca, 0x0b, 0xc9, 0x9c, 0x2e, 0xd7, 0xcc, 0x4a, 0x06, 0xc4, 0x81, 0x9d,
	0x51, 0x50, 0x33, 0xc2, 0xb7, 0x70, 0x7c, 0x4f, 0x85, 0x14, 0x7c, 0x41, 0x97, 0x7c, 0x37, 0x31,
	0x82, 0x86, 0xe6, 0x1b, 0x66, 0xa7, 0x6d, 0x12, 0xfb, 0x2e, 0x04, 0x5e, 0x65, 0xce, 0x74, 0xe4,
	0x5b, 0x27, 0x07, 0xf0, 0x25, 0xf4, 0xf6, 0x05, 0xb6, 0x43, 0x55, 0x6c, 0xaf, 0xce, 0xbe, 0x81,
	0x93, 0xb1, 0x7e, 0xd8, 0xee, 0x3e, 0x16, 0x53, 0x9a, 0xfd, 0x21, 0x23, 0x3c, 0x86, 0xd3, 0x6f,
	0x5d, 0x3b, 0x1b, 0x5e, 0x7c, 0xd8, 0x9e, 0x36, 0x71, 0xa0, 0x88, 0xcf, 0xd0, 0x6c, 0x22, 0x73,
	0x97, 0x41, 0x48, 0x4a, 0x78, 0xf5, 0xe1, 0x41, 0x38, 0x2d, 0x6f, 0x87, 0x46, 0xd0, 0xda, 0x86,
	0x89, 0xfa, 0x83, 0xea, 0x82, 0x83, 0xfd, 0xf3, 0xc5, 0xf1, 0xa1, 0x92, 0xf3, 0xc7, 0xff, 0xd0,
	0x23, 0x1c, 0xd5, 0x03, 0x40, 0x67, 0x35, 0xf6, 0x81, 0x68, 0xe3, 0xf3, 0x1f, 0xeb, 0x95, 0xe4,
	0x0c, 0xba, 0x5f, 0xf6, 0x45, 0x17, 0xb5, 0xae, 0xc3, 0x09, 0xc6, 0xf8, 0x37, 0x4a, 0xa9, 0x3d,
	0x6a, 0xcc, 0x7c, 0x35, 0x9f, 0xff, 0xb7, 0xb4, 0xeb, 0xcf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb9,
	0x40, 0xcf, 0x65, 0xd2, 0x02, 0x00, 0x00,
}
