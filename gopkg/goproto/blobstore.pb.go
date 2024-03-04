// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v4.25.3
// source: proto/blobstore.proto

package goproto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type BlobState int32

const (
	BlobState_INIT     BlobState = 0
	BlobState_PENDING  BlobState = 1
	BlobState_NORMAL   BlobState = 2
	BlobState_BROKEN   BlobState = 3
	BlobState_DELETING BlobState = 4
)

// Enum value maps for BlobState.
var (
	BlobState_name = map[int32]string{
		0: "INIT",
		1: "PENDING",
		2: "NORMAL",
		3: "BROKEN",
		4: "DELETING",
	}
	BlobState_value = map[string]int32{
		"INIT":     0,
		"PENDING":  1,
		"NORMAL":   2,
		"BROKEN":   3,
		"DELETING": 4,
	}
)

func (x BlobState) Enum() *BlobState {
	p := new(BlobState)
	*p = x
	return p
}

func (x BlobState) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (BlobState) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_blobstore_proto_enumTypes[0].Descriptor()
}

func (BlobState) Type() protoreflect.EnumType {
	return &file_proto_blobstore_proto_enumTypes[0]
}

func (x BlobState) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use BlobState.Descriptor instead.
func (BlobState) EnumDescriptor() ([]byte, []int) {
	return file_proto_blobstore_proto_rawDescGZIP(), []int{0}
}

type BlobInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VolumeId           uint64            `protobuf:"varint,1,opt,name=volume_id,json=volumeId,proto3" json:"volume_id,omitempty"`
	Seq                uint32            `protobuf:"varint,2,opt,name=seq,proto3" json:"seq,omitempty"`
	BlobSize           uint64            `protobuf:"varint,3,opt,name=blob_size,json=blobSize,proto3" json:"blob_size,omitempty"`
	State              BlobState         `protobuf:"varint,4,opt,name=state,proto3,enum=blobstore.BlobState" json:"state,omitempty"`
	CreateTimestamp    int64             `protobuf:"varint,5,opt,name=create_timestamp,json=createTimestamp,proto3" json:"create_timestamp,omitempty"` // also work as a version
	LastCheckTimestamp int64             `protobuf:"varint,6,opt,name=last_check_timestamp,json=lastCheckTimestamp,proto3" json:"last_check_timestamp,omitempty"`
	Meta               map[string]string `protobuf:"bytes,7,rep,name=meta,proto3" json:"meta,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *BlobInfo) Reset() {
	*x = BlobInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_blobstore_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BlobInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlobInfo) ProtoMessage() {}

func (x *BlobInfo) ProtoReflect() protoreflect.Message {
	mi := &file_proto_blobstore_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlobInfo.ProtoReflect.Descriptor instead.
func (*BlobInfo) Descriptor() ([]byte, []int) {
	return file_proto_blobstore_proto_rawDescGZIP(), []int{0}
}

func (x *BlobInfo) GetVolumeId() uint64 {
	if x != nil {
		return x.VolumeId
	}
	return 0
}

func (x *BlobInfo) GetSeq() uint32 {
	if x != nil {
		return x.Seq
	}
	return 0
}

func (x *BlobInfo) GetBlobSize() uint64 {
	if x != nil {
		return x.BlobSize
	}
	return 0
}

func (x *BlobInfo) GetState() BlobState {
	if x != nil {
		return x.State
	}
	return BlobState_INIT
}

func (x *BlobInfo) GetCreateTimestamp() int64 {
	if x != nil {
		return x.CreateTimestamp
	}
	return 0
}

func (x *BlobInfo) GetLastCheckTimestamp() int64 {
	if x != nil {
		return x.LastCheckTimestamp
	}
	return 0
}

func (x *BlobInfo) GetMeta() map[string]string {
	if x != nil {
		return x.Meta
	}
	return nil
}

type CreateBlobReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VolumeId uint64            `protobuf:"varint,1,opt,name=volume_id,json=volumeId,proto3" json:"volume_id,omitempty"`
	Seq      uint32            `protobuf:"varint,2,opt,name=seq,proto3" json:"seq,omitempty"`
	BlobSize uint64            `protobuf:"varint,3,opt,name=blob_size,json=blobSize,proto3" json:"blob_size,omitempty"`
	Meta     map[string]string `protobuf:"bytes,4,rep,name=meta,proto3" json:"meta,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *CreateBlobReq) Reset() {
	*x = CreateBlobReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_blobstore_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateBlobReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateBlobReq) ProtoMessage() {}

func (x *CreateBlobReq) ProtoReflect() protoreflect.Message {
	mi := &file_proto_blobstore_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateBlobReq.ProtoReflect.Descriptor instead.
func (*CreateBlobReq) Descriptor() ([]byte, []int) {
	return file_proto_blobstore_proto_rawDescGZIP(), []int{1}
}

func (x *CreateBlobReq) GetVolumeId() uint64 {
	if x != nil {
		return x.VolumeId
	}
	return 0
}

func (x *CreateBlobReq) GetSeq() uint32 {
	if x != nil {
		return x.Seq
	}
	return 0
}

func (x *CreateBlobReq) GetBlobSize() uint64 {
	if x != nil {
		return x.BlobSize
	}
	return 0
}

func (x *CreateBlobReq) GetMeta() map[string]string {
	if x != nil {
		return x.Meta
	}
	return nil
}

type BlobKey struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	VolumeId uint64 `protobuf:"varint,1,opt,name=volume_id,json=volumeId,proto3" json:"volume_id,omitempty"`
	Seq      uint32 `protobuf:"varint,2,opt,name=seq,proto3" json:"seq,omitempty"`
}

func (x *BlobKey) Reset() {
	*x = BlobKey{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_blobstore_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *BlobKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BlobKey) ProtoMessage() {}

func (x *BlobKey) ProtoReflect() protoreflect.Message {
	mi := &file_proto_blobstore_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BlobKey.ProtoReflect.Descriptor instead.
func (*BlobKey) Descriptor() ([]byte, []int) {
	return file_proto_blobstore_proto_rawDescGZIP(), []int{2}
}

func (x *BlobKey) GetVolumeId() uint64 {
	if x != nil {
		return x.VolumeId
	}
	return 0
}

func (x *BlobKey) GetSeq() uint32 {
	if x != nil {
		return x.Seq
	}
	return 0
}

type ListBlobReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Blob        *BlobKey   `protobuf:"bytes,1,opt,name=blob,proto3,oneof" json:"blob,omitempty"`
	StateFilter *BlobState `protobuf:"varint,2,opt,name=StateFilter,proto3,enum=blobstore.BlobState,oneof" json:"StateFilter,omitempty"`
	Limit       int64      `protobuf:"varint,3,opt,name=limit,proto3" json:"limit,omitempty"`
}

func (x *ListBlobReq) Reset() {
	*x = ListBlobReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_blobstore_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBlobReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBlobReq) ProtoMessage() {}

func (x *ListBlobReq) ProtoReflect() protoreflect.Message {
	mi := &file_proto_blobstore_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBlobReq.ProtoReflect.Descriptor instead.
func (*ListBlobReq) Descriptor() ([]byte, []int) {
	return file_proto_blobstore_proto_rawDescGZIP(), []int{3}
}

func (x *ListBlobReq) GetBlob() *BlobKey {
	if x != nil {
		return x.Blob
	}
	return nil
}

func (x *ListBlobReq) GetStateFilter() BlobState {
	if x != nil && x.StateFilter != nil {
		return *x.StateFilter
	}
	return BlobState_INIT
}

func (x *ListBlobReq) GetLimit() int64 {
	if x != nil {
		return x.Limit
	}
	return 0
}

type ListBlobResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BlobInfos             []*BlobInfo `protobuf:"bytes,1,rep,name=blob_infos,json=blobInfos,proto3" json:"blob_infos,omitempty"`
	NextContinuationToken *BlobKey    `protobuf:"bytes,2,opt,name=next_continuation_token,json=nextContinuationToken,proto3,oneof" json:"next_continuation_token,omitempty"`
	Truncated             bool        `protobuf:"varint,3,opt,name=truncated,proto3" json:"truncated,omitempty"`
}

func (x *ListBlobResp) Reset() {
	*x = ListBlobResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_blobstore_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListBlobResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListBlobResp) ProtoMessage() {}

func (x *ListBlobResp) ProtoReflect() protoreflect.Message {
	mi := &file_proto_blobstore_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListBlobResp.ProtoReflect.Descriptor instead.
func (*ListBlobResp) Descriptor() ([]byte, []int) {
	return file_proto_blobstore_proto_rawDescGZIP(), []int{4}
}

func (x *ListBlobResp) GetBlobInfos() []*BlobInfo {
	if x != nil {
		return x.BlobInfos
	}
	return nil
}

func (x *ListBlobResp) GetNextContinuationToken() *BlobKey {
	if x != nil {
		return x.NextContinuationToken
	}
	return nil
}

func (x *ListBlobResp) GetTruncated() bool {
	if x != nil {
		return x.Truncated
	}
	return false
}

var File_proto_blobstore_proto protoreflect.FileDescriptor

var file_proto_blobstore_proto_rawDesc = []byte{
	0x0a, 0x15, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f,
	0x72, 0x65, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0xcb, 0x02, 0x0a, 0x08, 0x42, 0x6c, 0x6f, 0x62, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x1b, 0x0a, 0x09,
	0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x08, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x49, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x65, 0x71,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x73, 0x65, 0x71, 0x12, 0x1b, 0x0a, 0x09, 0x62,
	0x6c, 0x6f, 0x62, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08,
	0x62, 0x6c, 0x6f, 0x62, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x2a, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x14, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74,
	0x6f, 0x72, 0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x05, 0x73,
	0x74, 0x61, 0x74, 0x65, 0x12, 0x29, 0x0a, 0x10, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x5f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x05, 0x20, 0x01, 0x28, 0x03, 0x52, 0x0f,
	0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x12,
	0x30, 0x0a, 0x14, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x63, 0x68, 0x65, 0x63, 0x6b, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x06, 0x20, 0x01, 0x28, 0x03, 0x52, 0x12, 0x6c,
	0x61, 0x73, 0x74, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x12, 0x31, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x07, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1d, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62,
	0x49, 0x6e, 0x66, 0x6f, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04,
	0x6d, 0x65, 0x74, 0x61, 0x1a, 0x37, 0x0a, 0x09, 0x4d, 0x65, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xcc, 0x01,
	0x0a, 0x0d, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65, 0x71, 0x12,
	0x1b, 0x0a, 0x09, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x08, 0x76, 0x6f, 0x6c, 0x75, 0x6d, 0x65, 0x49, 0x64, 0x12, 0x10, 0x0a, 0x03,
	0x73, 0x65, 0x71, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x73, 0x65, 0x71, 0x12, 0x1b,
	0x0a, 0x09, 0x62, 0x6c, 0x6f, 0x62, 0x5f, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x08, 0x62, 0x6c, 0x6f, 0x62, 0x53, 0x69, 0x7a, 0x65, 0x12, 0x36, 0x0a, 0x04, 0x6d,
	0x65, 0x74, 0x61, 0x18, 0x04, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x62, 0x6c, 0x6f, 0x62,
	0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x62,
	0x52, 0x65, 0x71, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x6d,
	0x65, 0x74, 0x61, 0x1a, 0x37, 0x0a, 0x09, 0x4d, 0x65, 0x74, 0x61, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0x38, 0x0a, 0x07,
	0x42, 0x6c, 0x6f, 0x62, 0x4b, 0x65, 0x79, 0x12, 0x1b, 0x0a, 0x09, 0x76, 0x6f, 0x6c, 0x75, 0x6d,
	0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x76, 0x6f, 0x6c, 0x75,
	0x6d, 0x65, 0x49, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x65, 0x71, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x03, 0x73, 0x65, 0x71, 0x22, 0xa6, 0x01, 0x0a, 0x0b, 0x4c, 0x69, 0x73, 0x74, 0x42,
	0x6c, 0x6f, 0x62, 0x52, 0x65, 0x71, 0x12, 0x2b, 0x0a, 0x04, 0x62, 0x6c, 0x6f, 0x62, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72, 0x65,
	0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x4b, 0x65, 0x79, 0x48, 0x00, 0x52, 0x04, 0x62, 0x6c, 0x6f, 0x62,
	0x88, 0x01, 0x01, 0x12, 0x3b, 0x0a, 0x0b, 0x53, 0x74, 0x61, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x74,
	0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x14, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73,
	0x74, 0x6f, 0x72, 0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x53, 0x74, 0x61, 0x74, 0x65, 0x48, 0x01,
	0x52, 0x0b, 0x53, 0x74, 0x61, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x88, 0x01, 0x01,
	0x12, 0x14, 0x0a, 0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x05, 0x6c, 0x69, 0x6d, 0x69, 0x74, 0x42, 0x07, 0x0a, 0x05, 0x5f, 0x62, 0x6c, 0x6f, 0x62, 0x42,
	0x0e, 0x0a, 0x0c, 0x5f, 0x53, 0x74, 0x61, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x22,
	0xcd, 0x01, 0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65, 0x73, 0x70,
	0x12, 0x32, 0x0a, 0x0a, 0x62, 0x6c, 0x6f, 0x62, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x73, 0x18, 0x01,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72, 0x65,
	0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x09, 0x62, 0x6c, 0x6f, 0x62, 0x49,
	0x6e, 0x66, 0x6f, 0x73, 0x12, 0x4f, 0x0a, 0x17, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x63, 0x6f, 0x6e,
	0x74, 0x69, 0x6e, 0x75, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x12, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72,
	0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x4b, 0x65, 0x79, 0x48, 0x00, 0x52, 0x15, 0x6e, 0x65, 0x78,
	0x74, 0x43, 0x6f, 0x6e, 0x74, 0x69, 0x6e, 0x75, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x6f, 0x6b,
	0x65, 0x6e, 0x88, 0x01, 0x01, 0x12, 0x1c, 0x0a, 0x09, 0x74, 0x72, 0x75, 0x6e, 0x63, 0x61, 0x74,
	0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x09, 0x74, 0x72, 0x75, 0x6e, 0x63, 0x61,
	0x74, 0x65, 0x64, 0x42, 0x1a, 0x0a, 0x18, 0x5f, 0x6e, 0x65, 0x78, 0x74, 0x5f, 0x63, 0x6f, 0x6e,
	0x74, 0x69, 0x6e, 0x75, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x2a,
	0x48, 0x0a, 0x09, 0x42, 0x6c, 0x6f, 0x62, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x08, 0x0a, 0x04,
	0x49, 0x4e, 0x49, 0x54, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x50, 0x45, 0x4e, 0x44, 0x49, 0x4e,
	0x47, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x4e, 0x4f, 0x52, 0x4d, 0x41, 0x4c, 0x10, 0x02, 0x12,
	0x0a, 0x0a, 0x06, 0x42, 0x52, 0x4f, 0x4b, 0x45, 0x4e, 0x10, 0x03, 0x12, 0x0c, 0x0a, 0x08, 0x44,
	0x45, 0x4c, 0x45, 0x54, 0x49, 0x4e, 0x47, 0x10, 0x04, 0x32, 0xf9, 0x01, 0x0a, 0x0b, 0x42, 0x6c,
	0x6f, 0x62, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x3e, 0x0a, 0x0a, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x62, 0x12, 0x18, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74,
	0x6f, 0x72, 0x65, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65,
	0x71, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x33, 0x0a, 0x08, 0x53, 0x74, 0x61,
	0x74, 0x42, 0x6c, 0x6f, 0x62, 0x12, 0x12, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72,
	0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x4b, 0x65, 0x79, 0x1a, 0x13, 0x2e, 0x62, 0x6c, 0x6f, 0x62,
	0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x38,
	0x0a, 0x0a, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x42, 0x6c, 0x6f, 0x62, 0x12, 0x12, 0x2e, 0x62,
	0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x42, 0x6c, 0x6f, 0x62, 0x4b, 0x65, 0x79,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x3b, 0x0a, 0x08, 0x4c, 0x69, 0x73, 0x74,
	0x42, 0x6c, 0x6f, 0x62, 0x12, 0x16, 0x2e, 0x62, 0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72, 0x65,
	0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6c, 0x6f, 0x62, 0x52, 0x65, 0x71, 0x1a, 0x17, 0x2e, 0x62,
	0x6c, 0x6f, 0x62, 0x73, 0x74, 0x6f, 0x72, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x42, 0x6c, 0x6f,
	0x62, 0x52, 0x65, 0x73, 0x70, 0x42, 0x0a, 0x5a, 0x08, 0x2f, 0x67, 0x6f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_blobstore_proto_rawDescOnce sync.Once
	file_proto_blobstore_proto_rawDescData = file_proto_blobstore_proto_rawDesc
)

func file_proto_blobstore_proto_rawDescGZIP() []byte {
	file_proto_blobstore_proto_rawDescOnce.Do(func() {
		file_proto_blobstore_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_blobstore_proto_rawDescData)
	})
	return file_proto_blobstore_proto_rawDescData
}

var file_proto_blobstore_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_proto_blobstore_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_proto_blobstore_proto_goTypes = []interface{}{
	(BlobState)(0),        // 0: blobstore.BlobState
	(*BlobInfo)(nil),      // 1: blobstore.BlobInfo
	(*CreateBlobReq)(nil), // 2: blobstore.CreateBlobReq
	(*BlobKey)(nil),       // 3: blobstore.BlobKey
	(*ListBlobReq)(nil),   // 4: blobstore.ListBlobReq
	(*ListBlobResp)(nil),  // 5: blobstore.ListBlobResp
	nil,                   // 6: blobstore.BlobInfo.MetaEntry
	nil,                   // 7: blobstore.CreateBlobReq.MetaEntry
	(*emptypb.Empty)(nil), // 8: google.protobuf.Empty
}
var file_proto_blobstore_proto_depIdxs = []int32{
	0,  // 0: blobstore.BlobInfo.state:type_name -> blobstore.BlobState
	6,  // 1: blobstore.BlobInfo.meta:type_name -> blobstore.BlobInfo.MetaEntry
	7,  // 2: blobstore.CreateBlobReq.meta:type_name -> blobstore.CreateBlobReq.MetaEntry
	3,  // 3: blobstore.ListBlobReq.blob:type_name -> blobstore.BlobKey
	0,  // 4: blobstore.ListBlobReq.StateFilter:type_name -> blobstore.BlobState
	1,  // 5: blobstore.ListBlobResp.blob_infos:type_name -> blobstore.BlobInfo
	3,  // 6: blobstore.ListBlobResp.next_continuation_token:type_name -> blobstore.BlobKey
	2,  // 7: blobstore.BlobService.CreateBlob:input_type -> blobstore.CreateBlobReq
	3,  // 8: blobstore.BlobService.StatBlob:input_type -> blobstore.BlobKey
	3,  // 9: blobstore.BlobService.DeleteBlob:input_type -> blobstore.BlobKey
	4,  // 10: blobstore.BlobService.ListBlob:input_type -> blobstore.ListBlobReq
	8,  // 11: blobstore.BlobService.CreateBlob:output_type -> google.protobuf.Empty
	1,  // 12: blobstore.BlobService.StatBlob:output_type -> blobstore.BlobInfo
	8,  // 13: blobstore.BlobService.DeleteBlob:output_type -> google.protobuf.Empty
	5,  // 14: blobstore.BlobService.ListBlob:output_type -> blobstore.ListBlobResp
	11, // [11:15] is the sub-list for method output_type
	7,  // [7:11] is the sub-list for method input_type
	7,  // [7:7] is the sub-list for extension type_name
	7,  // [7:7] is the sub-list for extension extendee
	0,  // [0:7] is the sub-list for field type_name
}

func init() { file_proto_blobstore_proto_init() }
func file_proto_blobstore_proto_init() {
	if File_proto_blobstore_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_blobstore_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BlobInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_blobstore_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CreateBlobReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_blobstore_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*BlobKey); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_blobstore_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBlobReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_blobstore_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListBlobResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_proto_blobstore_proto_msgTypes[3].OneofWrappers = []interface{}{}
	file_proto_blobstore_proto_msgTypes[4].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_blobstore_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_blobstore_proto_goTypes,
		DependencyIndexes: file_proto_blobstore_proto_depIdxs,
		EnumInfos:         file_proto_blobstore_proto_enumTypes,
		MessageInfos:      file_proto_blobstore_proto_msgTypes,
	}.Build()
	File_proto_blobstore_proto = out.File
	file_proto_blobstore_proto_rawDesc = nil
	file_proto_blobstore_proto_goTypes = nil
	file_proto_blobstore_proto_depIdxs = nil
}