// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v3.19.4
// source: usercenter.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// model
type User struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            int64                  `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
	Mobile        string                 `protobuf:"bytes,2,opt,name=mobile,proto3" json:"mobile"`
	Nickname      string                 `protobuf:"bytes,3,opt,name=nickname,proto3" json:"nickname"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *User) Reset() {
	*x = User{}
	mi := &file_usercenter_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use User.ProtoReflect.Descriptor instead.
func (*User) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{0}
}

func (x *User) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *User) GetMobile() string {
	if x != nil {
		return x.Mobile
	}
	return ""
}

func (x *User) GetNickname() string {
	if x != nil {
		return x.Nickname
	}
	return ""
}

// req 、resp
type RegisterReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Mobile        string                 `protobuf:"bytes,1,opt,name=mobile,proto3" json:"mobile"`
	Nickname      string                 `protobuf:"bytes,2,opt,name=nickname,proto3" json:"nickname"`
	Password      string                 `protobuf:"bytes,3,opt,name=password,proto3" json:"password"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RegisterReq) Reset() {
	*x = RegisterReq{}
	mi := &file_usercenter_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RegisterReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterReq) ProtoMessage() {}

func (x *RegisterReq) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterReq.ProtoReflect.Descriptor instead.
func (*RegisterReq) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{1}
}

func (x *RegisterReq) GetMobile() string {
	if x != nil {
		return x.Mobile
	}
	return ""
}

func (x *RegisterReq) GetNickname() string {
	if x != nil {
		return x.Nickname
	}
	return ""
}

func (x *RegisterReq) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type RegisterResp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AccessToken   string                 `protobuf:"bytes,1,opt,name=accessToken,proto3" json:"accessToken"`
	AccessExpire  int64                  `protobuf:"varint,2,opt,name=accessExpire,proto3" json:"accessExpire"`
	RefreshAfter  int64                  `protobuf:"varint,3,opt,name=refreshAfter,proto3" json:"refreshAfter"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RegisterResp) Reset() {
	*x = RegisterResp{}
	mi := &file_usercenter_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RegisterResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RegisterResp) ProtoMessage() {}

func (x *RegisterResp) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RegisterResp.ProtoReflect.Descriptor instead.
func (*RegisterResp) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{2}
}

func (x *RegisterResp) GetAccessToken() string {
	if x != nil {
		return x.AccessToken
	}
	return ""
}

func (x *RegisterResp) GetAccessExpire() int64 {
	if x != nil {
		return x.AccessExpire
	}
	return 0
}

func (x *RegisterResp) GetRefreshAfter() int64 {
	if x != nil {
		return x.RefreshAfter
	}
	return 0
}

type LoginReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Mobile        string                 `protobuf:"bytes,1,opt,name=mobile,proto3" json:"mobile"`
	Password      string                 `protobuf:"bytes,2,opt,name=password,proto3" json:"password"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoginReq) Reset() {
	*x = LoginReq{}
	mi := &file_usercenter_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoginReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginReq) ProtoMessage() {}

func (x *LoginReq) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginReq.ProtoReflect.Descriptor instead.
func (*LoginReq) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{3}
}

func (x *LoginReq) GetMobile() string {
	if x != nil {
		return x.Mobile
	}
	return ""
}

func (x *LoginReq) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type LoginResp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AccessToken   string                 `protobuf:"bytes,1,opt,name=accessToken,proto3" json:"accessToken"`
	AccessExpire  int64                  `protobuf:"varint,2,opt,name=accessExpire,proto3" json:"accessExpire"`
	RefreshAfter  int64                  `protobuf:"varint,3,opt,name=refreshAfter,proto3" json:"refreshAfter"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *LoginResp) Reset() {
	*x = LoginResp{}
	mi := &file_usercenter_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *LoginResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginResp) ProtoMessage() {}

func (x *LoginResp) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginResp.ProtoReflect.Descriptor instead.
func (*LoginResp) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{4}
}

func (x *LoginResp) GetAccessToken() string {
	if x != nil {
		return x.AccessToken
	}
	return ""
}

func (x *LoginResp) GetAccessExpire() int64 {
	if x != nil {
		return x.AccessExpire
	}
	return 0
}

func (x *LoginResp) GetRefreshAfter() int64 {
	if x != nil {
		return x.RefreshAfter
	}
	return 0
}

type GetUserInfoReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            int64                  `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetUserInfoReq) Reset() {
	*x = GetUserInfoReq{}
	mi := &file_usercenter_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetUserInfoReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUserInfoReq) ProtoMessage() {}

func (x *GetUserInfoReq) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUserInfoReq.ProtoReflect.Descriptor instead.
func (*GetUserInfoReq) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{5}
}

func (x *GetUserInfoReq) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type GetUserInfoResp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	User          *User                  `protobuf:"bytes,1,opt,name=user,proto3" json:"user"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetUserInfoResp) Reset() {
	*x = GetUserInfoResp{}
	mi := &file_usercenter_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetUserInfoResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUserInfoResp) ProtoMessage() {}

func (x *GetUserInfoResp) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUserInfoResp.ProtoReflect.Descriptor instead.
func (*GetUserInfoResp) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{6}
}

func (x *GetUserInfoResp) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}

type GenerateTokenReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        int64                  `protobuf:"varint,1,opt,name=userId,proto3" json:"userId"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GenerateTokenReq) Reset() {
	*x = GenerateTokenReq{}
	mi := &file_usercenter_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GenerateTokenReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenerateTokenReq) ProtoMessage() {}

func (x *GenerateTokenReq) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GenerateTokenReq.ProtoReflect.Descriptor instead.
func (*GenerateTokenReq) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{7}
}

func (x *GenerateTokenReq) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

type GenerateTokenResp struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	AccessToken   string                 `protobuf:"bytes,1,opt,name=accessToken,proto3" json:"accessToken"`
	AccessExpire  int64                  `protobuf:"varint,2,opt,name=accessExpire,proto3" json:"accessExpire"`
	RefreshAfter  int64                  `protobuf:"varint,3,opt,name=refreshAfter,proto3" json:"refreshAfter"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GenerateTokenResp) Reset() {
	*x = GenerateTokenResp{}
	mi := &file_usercenter_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GenerateTokenResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenerateTokenResp) ProtoMessage() {}

func (x *GenerateTokenResp) ProtoReflect() protoreflect.Message {
	mi := &file_usercenter_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GenerateTokenResp.ProtoReflect.Descriptor instead.
func (*GenerateTokenResp) Descriptor() ([]byte, []int) {
	return file_usercenter_proto_rawDescGZIP(), []int{8}
}

func (x *GenerateTokenResp) GetAccessToken() string {
	if x != nil {
		return x.AccessToken
	}
	return ""
}

func (x *GenerateTokenResp) GetAccessExpire() int64 {
	if x != nil {
		return x.AccessExpire
	}
	return 0
}

func (x *GenerateTokenResp) GetRefreshAfter() int64 {
	if x != nil {
		return x.RefreshAfter
	}
	return 0
}

var File_usercenter_proto protoreflect.FileDescriptor

const file_usercenter_proto_rawDesc = "" +
	"\n" +
	"\x10usercenter.proto\x12\x02pb\"J\n" +
	"\x04User\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\x03R\x02id\x12\x16\n" +
	"\x06mobile\x18\x02 \x01(\tR\x06mobile\x12\x1a\n" +
	"\bnickname\x18\x03 \x01(\tR\bnickname\"]\n" +
	"\vRegisterReq\x12\x16\n" +
	"\x06mobile\x18\x01 \x01(\tR\x06mobile\x12\x1a\n" +
	"\bnickname\x18\x02 \x01(\tR\bnickname\x12\x1a\n" +
	"\bpassword\x18\x03 \x01(\tR\bpassword\"x\n" +
	"\fRegisterResp\x12 \n" +
	"\vaccessToken\x18\x01 \x01(\tR\vaccessToken\x12\"\n" +
	"\faccessExpire\x18\x02 \x01(\x03R\faccessExpire\x12\"\n" +
	"\frefreshAfter\x18\x03 \x01(\x03R\frefreshAfter\">\n" +
	"\bLoginReq\x12\x16\n" +
	"\x06mobile\x18\x01 \x01(\tR\x06mobile\x12\x1a\n" +
	"\bpassword\x18\x02 \x01(\tR\bpassword\"u\n" +
	"\tLoginResp\x12 \n" +
	"\vaccessToken\x18\x01 \x01(\tR\vaccessToken\x12\"\n" +
	"\faccessExpire\x18\x02 \x01(\x03R\faccessExpire\x12\"\n" +
	"\frefreshAfter\x18\x03 \x01(\x03R\frefreshAfter\" \n" +
	"\x0eGetUserInfoReq\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\x03R\x02id\"/\n" +
	"\x0fGetUserInfoResp\x12\x1c\n" +
	"\x04user\x18\x01 \x01(\v2\b.pb.UserR\x04user\"*\n" +
	"\x10GenerateTokenReq\x12\x16\n" +
	"\x06userId\x18\x01 \x01(\x03R\x06userId\"}\n" +
	"\x11GenerateTokenResp\x12 \n" +
	"\vaccessToken\x18\x01 \x01(\tR\vaccessToken\x12\"\n" +
	"\faccessExpire\x18\x02 \x01(\x03R\faccessExpire\x12\"\n" +
	"\frefreshAfter\x18\x03 \x01(\x03R\frefreshAfter2\xd7\x01\n" +
	"\n" +
	"usercenter\x12$\n" +
	"\x05login\x12\f.pb.LoginReq\x1a\r.pb.LoginResp\x12-\n" +
	"\bregister\x12\x0f.pb.RegisterReq\x1a\x10.pb.RegisterResp\x126\n" +
	"\vgetUserInfo\x12\x12.pb.GetUserInfoReq\x1a\x13.pb.GetUserInfoResp\x12<\n" +
	"\rgenerateToken\x12\x14.pb.GenerateTokenReq\x1a\x15.pb.GenerateTokenRespB\x06Z\x04./pbb\x06proto3"

var (
	file_usercenter_proto_rawDescOnce sync.Once
	file_usercenter_proto_rawDescData []byte
)

func file_usercenter_proto_rawDescGZIP() []byte {
	file_usercenter_proto_rawDescOnce.Do(func() {
		file_usercenter_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_usercenter_proto_rawDesc), len(file_usercenter_proto_rawDesc)))
	})
	return file_usercenter_proto_rawDescData
}

var file_usercenter_proto_msgTypes = make([]protoimpl.MessageInfo, 9)
var file_usercenter_proto_goTypes = []any{
	(*User)(nil),              // 0: pb.User
	(*RegisterReq)(nil),       // 1: pb.RegisterReq
	(*RegisterResp)(nil),      // 2: pb.RegisterResp
	(*LoginReq)(nil),          // 3: pb.LoginReq
	(*LoginResp)(nil),         // 4: pb.LoginResp
	(*GetUserInfoReq)(nil),    // 5: pb.GetUserInfoReq
	(*GetUserInfoResp)(nil),   // 6: pb.GetUserInfoResp
	(*GenerateTokenReq)(nil),  // 7: pb.GenerateTokenReq
	(*GenerateTokenResp)(nil), // 8: pb.GenerateTokenResp
}
var file_usercenter_proto_depIdxs = []int32{
	0, // 0: pb.GetUserInfoResp.user:type_name -> pb.User
	3, // 1: pb.usercenter.login:input_type -> pb.LoginReq
	1, // 2: pb.usercenter.register:input_type -> pb.RegisterReq
	5, // 3: pb.usercenter.getUserInfo:input_type -> pb.GetUserInfoReq
	7, // 4: pb.usercenter.generateToken:input_type -> pb.GenerateTokenReq
	4, // 5: pb.usercenter.login:output_type -> pb.LoginResp
	2, // 6: pb.usercenter.register:output_type -> pb.RegisterResp
	6, // 7: pb.usercenter.getUserInfo:output_type -> pb.GetUserInfoResp
	8, // 8: pb.usercenter.generateToken:output_type -> pb.GenerateTokenResp
	5, // [5:9] is the sub-list for method output_type
	1, // [1:5] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_usercenter_proto_init() }
func file_usercenter_proto_init() {
	if File_usercenter_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_usercenter_proto_rawDesc), len(file_usercenter_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   9,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_usercenter_proto_goTypes,
		DependencyIndexes: file_usercenter_proto_depIdxs,
		MessageInfos:      file_usercenter_proto_msgTypes,
	}.Build()
	File_usercenter_proto = out.File
	file_usercenter_proto_goTypes = nil
	file_usercenter_proto_depIdxs = nil
}
