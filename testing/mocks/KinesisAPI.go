package mocks

import "github.com/stretchr/testify/mock"

import "github.com/aws/aws-sdk-go/aws/request"
import "github.com/aws/aws-sdk-go/service/kinesis"

// AUTO-GENERATED MOCK. DO NOT EDIT.
// USE make mocks TO REGENERATE.

type KinesisAPI struct {
	mock.Mock
}

func (m *KinesisAPI) Name_AddTagsToStreamRequest() string {
	return "AddTagsToStreamRequest"
}
func (m *KinesisAPI) MockOn_AddTagsToStreamRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("AddTagsToStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_AddTagsToStreamRequest(_a0 *kinesis.AddTagsToStreamInput) *mock.Mock {
	return m.Mock.On("AddTagsToStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_AddTagsToStreamRequest() *mock.Mock {
	return m.Mock.On("AddTagsToStreamRequest", mock.Anything)
}
func (m *KinesisAPI) AddTagsToStreamRequest(_a0 *kinesis.AddTagsToStreamInput) (*request.Request, *kinesis.AddTagsToStreamOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.AddTagsToStreamInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.AddTagsToStreamOutput
	if rf, ok := ret.Get(1).(func(*kinesis.AddTagsToStreamInput) *kinesis.AddTagsToStreamOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.AddTagsToStreamOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_AddTagsToStream() string {
	return "AddTagsToStream"
}
func (m *KinesisAPI) MockOn_AddTagsToStream(_a0 interface{}) *mock.Mock {
	return m.Mock.On("AddTagsToStream", _a0)
}
func (m *KinesisAPI) MockOnTyped_AddTagsToStream(_a0 *kinesis.AddTagsToStreamInput) *mock.Mock {
	return m.Mock.On("AddTagsToStream", _a0)
}
func (m *KinesisAPI) MockOnAny_AddTagsToStream() *mock.Mock {
	return m.Mock.On("AddTagsToStream", mock.Anything)
}
func (m *KinesisAPI) AddTagsToStream(_a0 *kinesis.AddTagsToStreamInput) (*kinesis.AddTagsToStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.AddTagsToStreamOutput
	if rf, ok := ret.Get(0).(func(*kinesis.AddTagsToStreamInput) *kinesis.AddTagsToStreamOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.AddTagsToStreamOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.AddTagsToStreamInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_CreateStreamRequest() string {
	return "CreateStreamRequest"
}
func (m *KinesisAPI) MockOn_CreateStreamRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CreateStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_CreateStreamRequest(_a0 *kinesis.CreateStreamInput) *mock.Mock {
	return m.Mock.On("CreateStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_CreateStreamRequest() *mock.Mock {
	return m.Mock.On("CreateStreamRequest", mock.Anything)
}
func (m *KinesisAPI) CreateStreamRequest(_a0 *kinesis.CreateStreamInput) (*request.Request, *kinesis.CreateStreamOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.CreateStreamInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.CreateStreamOutput
	if rf, ok := ret.Get(1).(func(*kinesis.CreateStreamInput) *kinesis.CreateStreamOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.CreateStreamOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_CreateStream() string {
	return "CreateStream"
}
func (m *KinesisAPI) MockOn_CreateStream(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CreateStream", _a0)
}
func (m *KinesisAPI) MockOnTyped_CreateStream(_a0 *kinesis.CreateStreamInput) *mock.Mock {
	return m.Mock.On("CreateStream", _a0)
}
func (m *KinesisAPI) MockOnAny_CreateStream() *mock.Mock {
	return m.Mock.On("CreateStream", mock.Anything)
}
func (m *KinesisAPI) CreateStream(_a0 *kinesis.CreateStreamInput) (*kinesis.CreateStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.CreateStreamOutput
	if rf, ok := ret.Get(0).(func(*kinesis.CreateStreamInput) *kinesis.CreateStreamOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.CreateStreamOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.CreateStreamInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_DeleteStreamRequest() string {
	return "DeleteStreamRequest"
}
func (m *KinesisAPI) MockOn_DeleteStreamRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeleteStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_DeleteStreamRequest(_a0 *kinesis.DeleteStreamInput) *mock.Mock {
	return m.Mock.On("DeleteStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_DeleteStreamRequest() *mock.Mock {
	return m.Mock.On("DeleteStreamRequest", mock.Anything)
}
func (m *KinesisAPI) DeleteStreamRequest(_a0 *kinesis.DeleteStreamInput) (*request.Request, *kinesis.DeleteStreamOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.DeleteStreamInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.DeleteStreamOutput
	if rf, ok := ret.Get(1).(func(*kinesis.DeleteStreamInput) *kinesis.DeleteStreamOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.DeleteStreamOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_DeleteStream() string {
	return "DeleteStream"
}
func (m *KinesisAPI) MockOn_DeleteStream(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeleteStream", _a0)
}
func (m *KinesisAPI) MockOnTyped_DeleteStream(_a0 *kinesis.DeleteStreamInput) *mock.Mock {
	return m.Mock.On("DeleteStream", _a0)
}
func (m *KinesisAPI) MockOnAny_DeleteStream() *mock.Mock {
	return m.Mock.On("DeleteStream", mock.Anything)
}
func (m *KinesisAPI) DeleteStream(_a0 *kinesis.DeleteStreamInput) (*kinesis.DeleteStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.DeleteStreamOutput
	if rf, ok := ret.Get(0).(func(*kinesis.DeleteStreamInput) *kinesis.DeleteStreamOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.DeleteStreamOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.DeleteStreamInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_DescribeStreamRequest() string {
	return "DescribeStreamRequest"
}
func (m *KinesisAPI) MockOn_DescribeStreamRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_DescribeStreamRequest(_a0 *kinesis.DescribeStreamInput) *mock.Mock {
	return m.Mock.On("DescribeStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_DescribeStreamRequest() *mock.Mock {
	return m.Mock.On("DescribeStreamRequest", mock.Anything)
}
func (m *KinesisAPI) DescribeStreamRequest(_a0 *kinesis.DescribeStreamInput) (*request.Request, *kinesis.DescribeStreamOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.DescribeStreamInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.DescribeStreamOutput
	if rf, ok := ret.Get(1).(func(*kinesis.DescribeStreamInput) *kinesis.DescribeStreamOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.DescribeStreamOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_DescribeStream() string {
	return "DescribeStream"
}
func (m *KinesisAPI) MockOn_DescribeStream(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeStream", _a0)
}
func (m *KinesisAPI) MockOnTyped_DescribeStream(_a0 *kinesis.DescribeStreamInput) *mock.Mock {
	return m.Mock.On("DescribeStream", _a0)
}
func (m *KinesisAPI) MockOnAny_DescribeStream() *mock.Mock {
	return m.Mock.On("DescribeStream", mock.Anything)
}
func (m *KinesisAPI) DescribeStream(_a0 *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.DescribeStreamOutput
	if rf, ok := ret.Get(0).(func(*kinesis.DescribeStreamInput) *kinesis.DescribeStreamOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.DescribeStreamOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.DescribeStreamInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_DescribeStreamPages() string {
	return "DescribeStreamPages"
}
func (m *KinesisAPI) MockOn_DescribeStreamPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("DescribeStreamPages", _a0, _a1)
}
func (m *KinesisAPI) MockOnTyped_DescribeStreamPages(_a0 *kinesis.DescribeStreamInput, _a1 func(*kinesis.DescribeStreamOutput, bool) bool) *mock.Mock {
	return m.Mock.On("DescribeStreamPages", _a0, _a1)
}
func (m *KinesisAPI) MockOnAny_DescribeStreamPages() *mock.Mock {
	return m.Mock.On("DescribeStreamPages", mock.Anything, mock.Anything)
}
func (m *KinesisAPI) DescribeStreamPages(_a0 *kinesis.DescribeStreamInput, _a1 func(*kinesis.DescribeStreamOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*kinesis.DescribeStreamInput, func(*kinesis.DescribeStreamOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *KinesisAPI) Name_GetRecordsRequest() string {
	return "GetRecordsRequest"
}
func (m *KinesisAPI) MockOn_GetRecordsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("GetRecordsRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_GetRecordsRequest(_a0 *kinesis.GetRecordsInput) *mock.Mock {
	return m.Mock.On("GetRecordsRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_GetRecordsRequest() *mock.Mock {
	return m.Mock.On("GetRecordsRequest", mock.Anything)
}
func (m *KinesisAPI) GetRecordsRequest(_a0 *kinesis.GetRecordsInput) (*request.Request, *kinesis.GetRecordsOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.GetRecordsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.GetRecordsOutput
	if rf, ok := ret.Get(1).(func(*kinesis.GetRecordsInput) *kinesis.GetRecordsOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.GetRecordsOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_GetRecords() string {
	return "GetRecords"
}
func (m *KinesisAPI) MockOn_GetRecords(_a0 interface{}) *mock.Mock {
	return m.Mock.On("GetRecords", _a0)
}
func (m *KinesisAPI) MockOnTyped_GetRecords(_a0 *kinesis.GetRecordsInput) *mock.Mock {
	return m.Mock.On("GetRecords", _a0)
}
func (m *KinesisAPI) MockOnAny_GetRecords() *mock.Mock {
	return m.Mock.On("GetRecords", mock.Anything)
}
func (m *KinesisAPI) GetRecords(_a0 *kinesis.GetRecordsInput) (*kinesis.GetRecordsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.GetRecordsOutput
	if rf, ok := ret.Get(0).(func(*kinesis.GetRecordsInput) *kinesis.GetRecordsOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.GetRecordsOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.GetRecordsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_GetShardIteratorRequest() string {
	return "GetShardIteratorRequest"
}
func (m *KinesisAPI) MockOn_GetShardIteratorRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("GetShardIteratorRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_GetShardIteratorRequest(_a0 *kinesis.GetShardIteratorInput) *mock.Mock {
	return m.Mock.On("GetShardIteratorRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_GetShardIteratorRequest() *mock.Mock {
	return m.Mock.On("GetShardIteratorRequest", mock.Anything)
}
func (m *KinesisAPI) GetShardIteratorRequest(_a0 *kinesis.GetShardIteratorInput) (*request.Request, *kinesis.GetShardIteratorOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.GetShardIteratorInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.GetShardIteratorOutput
	if rf, ok := ret.Get(1).(func(*kinesis.GetShardIteratorInput) *kinesis.GetShardIteratorOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.GetShardIteratorOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_GetShardIterator() string {
	return "GetShardIterator"
}
func (m *KinesisAPI) MockOn_GetShardIterator(_a0 interface{}) *mock.Mock {
	return m.Mock.On("GetShardIterator", _a0)
}
func (m *KinesisAPI) MockOnTyped_GetShardIterator(_a0 *kinesis.GetShardIteratorInput) *mock.Mock {
	return m.Mock.On("GetShardIterator", _a0)
}
func (m *KinesisAPI) MockOnAny_GetShardIterator() *mock.Mock {
	return m.Mock.On("GetShardIterator", mock.Anything)
}
func (m *KinesisAPI) GetShardIterator(_a0 *kinesis.GetShardIteratorInput) (*kinesis.GetShardIteratorOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.GetShardIteratorOutput
	if rf, ok := ret.Get(0).(func(*kinesis.GetShardIteratorInput) *kinesis.GetShardIteratorOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.GetShardIteratorOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.GetShardIteratorInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_ListStreamsRequest() string {
	return "ListStreamsRequest"
}
func (m *KinesisAPI) MockOn_ListStreamsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListStreamsRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_ListStreamsRequest(_a0 *kinesis.ListStreamsInput) *mock.Mock {
	return m.Mock.On("ListStreamsRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_ListStreamsRequest() *mock.Mock {
	return m.Mock.On("ListStreamsRequest", mock.Anything)
}
func (m *KinesisAPI) ListStreamsRequest(_a0 *kinesis.ListStreamsInput) (*request.Request, *kinesis.ListStreamsOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.ListStreamsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.ListStreamsOutput
	if rf, ok := ret.Get(1).(func(*kinesis.ListStreamsInput) *kinesis.ListStreamsOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.ListStreamsOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_ListStreams() string {
	return "ListStreams"
}
func (m *KinesisAPI) MockOn_ListStreams(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListStreams", _a0)
}
func (m *KinesisAPI) MockOnTyped_ListStreams(_a0 *kinesis.ListStreamsInput) *mock.Mock {
	return m.Mock.On("ListStreams", _a0)
}
func (m *KinesisAPI) MockOnAny_ListStreams() *mock.Mock {
	return m.Mock.On("ListStreams", mock.Anything)
}
func (m *KinesisAPI) ListStreams(_a0 *kinesis.ListStreamsInput) (*kinesis.ListStreamsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.ListStreamsOutput
	if rf, ok := ret.Get(0).(func(*kinesis.ListStreamsInput) *kinesis.ListStreamsOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.ListStreamsOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.ListStreamsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_ListStreamsPages() string {
	return "ListStreamsPages"
}
func (m *KinesisAPI) MockOn_ListStreamsPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("ListStreamsPages", _a0, _a1)
}
func (m *KinesisAPI) MockOnTyped_ListStreamsPages(_a0 *kinesis.ListStreamsInput, _a1 func(*kinesis.ListStreamsOutput, bool) bool) *mock.Mock {
	return m.Mock.On("ListStreamsPages", _a0, _a1)
}
func (m *KinesisAPI) MockOnAny_ListStreamsPages() *mock.Mock {
	return m.Mock.On("ListStreamsPages", mock.Anything, mock.Anything)
}
func (m *KinesisAPI) ListStreamsPages(_a0 *kinesis.ListStreamsInput, _a1 func(*kinesis.ListStreamsOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*kinesis.ListStreamsInput, func(*kinesis.ListStreamsOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *KinesisAPI) Name_ListTagsForStreamRequest() string {
	return "ListTagsForStreamRequest"
}
func (m *KinesisAPI) MockOn_ListTagsForStreamRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListTagsForStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_ListTagsForStreamRequest(_a0 *kinesis.ListTagsForStreamInput) *mock.Mock {
	return m.Mock.On("ListTagsForStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_ListTagsForStreamRequest() *mock.Mock {
	return m.Mock.On("ListTagsForStreamRequest", mock.Anything)
}
func (m *KinesisAPI) ListTagsForStreamRequest(_a0 *kinesis.ListTagsForStreamInput) (*request.Request, *kinesis.ListTagsForStreamOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.ListTagsForStreamInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.ListTagsForStreamOutput
	if rf, ok := ret.Get(1).(func(*kinesis.ListTagsForStreamInput) *kinesis.ListTagsForStreamOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.ListTagsForStreamOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_ListTagsForStream() string {
	return "ListTagsForStream"
}
func (m *KinesisAPI) MockOn_ListTagsForStream(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListTagsForStream", _a0)
}
func (m *KinesisAPI) MockOnTyped_ListTagsForStream(_a0 *kinesis.ListTagsForStreamInput) *mock.Mock {
	return m.Mock.On("ListTagsForStream", _a0)
}
func (m *KinesisAPI) MockOnAny_ListTagsForStream() *mock.Mock {
	return m.Mock.On("ListTagsForStream", mock.Anything)
}
func (m *KinesisAPI) ListTagsForStream(_a0 *kinesis.ListTagsForStreamInput) (*kinesis.ListTagsForStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.ListTagsForStreamOutput
	if rf, ok := ret.Get(0).(func(*kinesis.ListTagsForStreamInput) *kinesis.ListTagsForStreamOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.ListTagsForStreamOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.ListTagsForStreamInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_MergeShardsRequest() string {
	return "MergeShardsRequest"
}
func (m *KinesisAPI) MockOn_MergeShardsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("MergeShardsRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_MergeShardsRequest(_a0 *kinesis.MergeShardsInput) *mock.Mock {
	return m.Mock.On("MergeShardsRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_MergeShardsRequest() *mock.Mock {
	return m.Mock.On("MergeShardsRequest", mock.Anything)
}
func (m *KinesisAPI) MergeShardsRequest(_a0 *kinesis.MergeShardsInput) (*request.Request, *kinesis.MergeShardsOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.MergeShardsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.MergeShardsOutput
	if rf, ok := ret.Get(1).(func(*kinesis.MergeShardsInput) *kinesis.MergeShardsOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.MergeShardsOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_MergeShards() string {
	return "MergeShards"
}
func (m *KinesisAPI) MockOn_MergeShards(_a0 interface{}) *mock.Mock {
	return m.Mock.On("MergeShards", _a0)
}
func (m *KinesisAPI) MockOnTyped_MergeShards(_a0 *kinesis.MergeShardsInput) *mock.Mock {
	return m.Mock.On("MergeShards", _a0)
}
func (m *KinesisAPI) MockOnAny_MergeShards() *mock.Mock {
	return m.Mock.On("MergeShards", mock.Anything)
}
func (m *KinesisAPI) MergeShards(_a0 *kinesis.MergeShardsInput) (*kinesis.MergeShardsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.MergeShardsOutput
	if rf, ok := ret.Get(0).(func(*kinesis.MergeShardsInput) *kinesis.MergeShardsOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.MergeShardsOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.MergeShardsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_PutRecordRequest() string {
	return "PutRecordRequest"
}
func (m *KinesisAPI) MockOn_PutRecordRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PutRecordRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_PutRecordRequest(_a0 *kinesis.PutRecordInput) *mock.Mock {
	return m.Mock.On("PutRecordRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_PutRecordRequest() *mock.Mock {
	return m.Mock.On("PutRecordRequest", mock.Anything)
}
func (m *KinesisAPI) PutRecordRequest(_a0 *kinesis.PutRecordInput) (*request.Request, *kinesis.PutRecordOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.PutRecordInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.PutRecordOutput
	if rf, ok := ret.Get(1).(func(*kinesis.PutRecordInput) *kinesis.PutRecordOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.PutRecordOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_PutRecord() string {
	return "PutRecord"
}
func (m *KinesisAPI) MockOn_PutRecord(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PutRecord", _a0)
}
func (m *KinesisAPI) MockOnTyped_PutRecord(_a0 *kinesis.PutRecordInput) *mock.Mock {
	return m.Mock.On("PutRecord", _a0)
}
func (m *KinesisAPI) MockOnAny_PutRecord() *mock.Mock {
	return m.Mock.On("PutRecord", mock.Anything)
}
func (m *KinesisAPI) PutRecord(_a0 *kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.PutRecordOutput
	if rf, ok := ret.Get(0).(func(*kinesis.PutRecordInput) *kinesis.PutRecordOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.PutRecordOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.PutRecordInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_PutRecordsRequest() string {
	return "PutRecordsRequest"
}
func (m *KinesisAPI) MockOn_PutRecordsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PutRecordsRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_PutRecordsRequest(_a0 *kinesis.PutRecordsInput) *mock.Mock {
	return m.Mock.On("PutRecordsRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_PutRecordsRequest() *mock.Mock {
	return m.Mock.On("PutRecordsRequest", mock.Anything)
}
func (m *KinesisAPI) PutRecordsRequest(_a0 *kinesis.PutRecordsInput) (*request.Request, *kinesis.PutRecordsOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.PutRecordsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.PutRecordsOutput
	if rf, ok := ret.Get(1).(func(*kinesis.PutRecordsInput) *kinesis.PutRecordsOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.PutRecordsOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_PutRecords() string {
	return "PutRecords"
}
func (m *KinesisAPI) MockOn_PutRecords(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PutRecords", _a0)
}
func (m *KinesisAPI) MockOnTyped_PutRecords(_a0 *kinesis.PutRecordsInput) *mock.Mock {
	return m.Mock.On("PutRecords", _a0)
}
func (m *KinesisAPI) MockOnAny_PutRecords() *mock.Mock {
	return m.Mock.On("PutRecords", mock.Anything)
}
func (m *KinesisAPI) PutRecords(_a0 *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.PutRecordsOutput
	if rf, ok := ret.Get(0).(func(*kinesis.PutRecordsInput) *kinesis.PutRecordsOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.PutRecordsOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.PutRecordsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_RemoveTagsFromStreamRequest() string {
	return "RemoveTagsFromStreamRequest"
}
func (m *KinesisAPI) MockOn_RemoveTagsFromStreamRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RemoveTagsFromStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_RemoveTagsFromStreamRequest(_a0 *kinesis.RemoveTagsFromStreamInput) *mock.Mock {
	return m.Mock.On("RemoveTagsFromStreamRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_RemoveTagsFromStreamRequest() *mock.Mock {
	return m.Mock.On("RemoveTagsFromStreamRequest", mock.Anything)
}
func (m *KinesisAPI) RemoveTagsFromStreamRequest(_a0 *kinesis.RemoveTagsFromStreamInput) (*request.Request, *kinesis.RemoveTagsFromStreamOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.RemoveTagsFromStreamInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.RemoveTagsFromStreamOutput
	if rf, ok := ret.Get(1).(func(*kinesis.RemoveTagsFromStreamInput) *kinesis.RemoveTagsFromStreamOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.RemoveTagsFromStreamOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_RemoveTagsFromStream() string {
	return "RemoveTagsFromStream"
}
func (m *KinesisAPI) MockOn_RemoveTagsFromStream(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RemoveTagsFromStream", _a0)
}
func (m *KinesisAPI) MockOnTyped_RemoveTagsFromStream(_a0 *kinesis.RemoveTagsFromStreamInput) *mock.Mock {
	return m.Mock.On("RemoveTagsFromStream", _a0)
}
func (m *KinesisAPI) MockOnAny_RemoveTagsFromStream() *mock.Mock {
	return m.Mock.On("RemoveTagsFromStream", mock.Anything)
}
func (m *KinesisAPI) RemoveTagsFromStream(_a0 *kinesis.RemoveTagsFromStreamInput) (*kinesis.RemoveTagsFromStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.RemoveTagsFromStreamOutput
	if rf, ok := ret.Get(0).(func(*kinesis.RemoveTagsFromStreamInput) *kinesis.RemoveTagsFromStreamOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.RemoveTagsFromStreamOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.RemoveTagsFromStreamInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *KinesisAPI) Name_SplitShardRequest() string {
	return "SplitShardRequest"
}
func (m *KinesisAPI) MockOn_SplitShardRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("SplitShardRequest", _a0)
}
func (m *KinesisAPI) MockOnTyped_SplitShardRequest(_a0 *kinesis.SplitShardInput) *mock.Mock {
	return m.Mock.On("SplitShardRequest", _a0)
}
func (m *KinesisAPI) MockOnAny_SplitShardRequest() *mock.Mock {
	return m.Mock.On("SplitShardRequest", mock.Anything)
}
func (m *KinesisAPI) SplitShardRequest(_a0 *kinesis.SplitShardInput) (*request.Request, *kinesis.SplitShardOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*kinesis.SplitShardInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *kinesis.SplitShardOutput
	if rf, ok := ret.Get(1).(func(*kinesis.SplitShardInput) *kinesis.SplitShardOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*kinesis.SplitShardOutput)
		}
	}

	return r0, r1
}
func (m *KinesisAPI) Name_SplitShard() string {
	return "SplitShard"
}
func (m *KinesisAPI) MockOn_SplitShard(_a0 interface{}) *mock.Mock {
	return m.Mock.On("SplitShard", _a0)
}
func (m *KinesisAPI) MockOnTyped_SplitShard(_a0 *kinesis.SplitShardInput) *mock.Mock {
	return m.Mock.On("SplitShard", _a0)
}
func (m *KinesisAPI) MockOnAny_SplitShard() *mock.Mock {
	return m.Mock.On("SplitShard", mock.Anything)
}
func (m *KinesisAPI) SplitShard(_a0 *kinesis.SplitShardInput) (*kinesis.SplitShardOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.SplitShardOutput
	if rf, ok := ret.Get(0).(func(*kinesis.SplitShardInput) *kinesis.SplitShardOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*kinesis.SplitShardOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*kinesis.SplitShardInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
