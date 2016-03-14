package s3serializer

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	bucket = "the-bucket"
	prefix = "the-prefix"

	staticKey = "a-new-key"
)

var (
	staticKeyGen = func() string { return staticKey }
)

func TestSerialize_SmallData(t *testing.T) {
	defer func() {
		minS3Length = defaultMinS3Length
		keyGen = defaultKeyGen
	}()
	keyGen = staticKeyGen
	minS3Length = 32000

	s3c := &fakeS3{}
	under := &fakeSerializer{}
	ss := New(s3c, bucket, prefix, under)

	td := strings.Repeat("x", 5)

	ser, err := ss.Serialize(td)

	if err != nil {
		t.Fatalf("expected nil err, got %q", err)
	}

	if s3c.put.input != nil {
		t.Fatalf("expected no S3 request, got %+v", s3c.put.input)
	}

	if !reflect.DeepEqual(under.sReq, td) {
		t.Fatalf("expected under to serialize %+v, got %+v", td, under.sReq)
	}

	if ser != td {
		t.Fatalf("expected direct under serialization, got %q", ser)
	}
}

func TestSerialize_LargeData(t *testing.T) {
	defer func() { keyGen = defaultKeyGen }()
	keyGen = staticKeyGen

	s3c := &fakeS3{}
	under := &fakeSerializer{}
	ss := New(s3c, bucket, prefix, under)

	td := strings.Repeat("x", 64000)

	ser, err := ss.Serialize(td)

	if err != nil {
		t.Fatalf("expected nil err, got %q", err)
	}

	if s3c.put.input == nil {
		t.Fatalf("expected S3 request")
	}

	if want, got := *s3c.put.input.Bucket, bucket; want != got {
		t.Fatalf("expected S3 request for bucket %q, got %q", want, got)
	}

	if want, got := *s3c.put.input.Key, prefix+"/"+staticKey; want != got {
		t.Fatalf("expected S3 request for key %q, got %q", want, got)
	}

	if want, got := td, s3c.put.body; want != got {
		t.Fatalf("expected S3 request with body %q, got %q", want, got)
	}

	if !reflect.DeepEqual(under.sReq, td) {
		t.Fatalf("expected under to serialize %+v, got %+v", td, under.sReq)
	}

	if !strings.HasPrefix(ser, magicPrefix) {
		t.Fatalf("expected %q to begin with magicPrefix %q", ser, magicPrefix)
	}
}

func TestDeserialize_NoMagic(t *testing.T) {
	s3c := &fakeS3{}
	under := &fakeSerializer{}
	ss := New(s3c, bucket, prefix, under)

	td := strings.Repeat("x", 5)
	under.dRes = td

	var deser string

	err := ss.Deserialize(td, &deser)

	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(td, deser) {
		t.Fatalf("expected deser to be %q, got %v", td, deser)
	}

	if s3c.get.input != nil {
		t.Fatalf("expected no S3 request, got %+v", s3c.get.input)
	}
}

func TestDeserialize_Magic(t *testing.T) {
	s3c := &fakeS3{}
	under := &fakeSerializer{}
	ss := New(s3c, bucket, prefix, under)

	td := strings.Repeat("x", 5)
	under.dRes = td

	s3c.get.body = td

	var deser string

	err := ss.Deserialize("s3+ser://"+bucket+"/"+prefix+"/some-key", &deser)

	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(td, deser) {
		t.Fatalf("expected deser to be %q, got %v", td, deser)
	}

	if s3c.get.input == nil {
		t.Fatalf("expected S3 request")
	}

	if want, got := bucket, *s3c.get.input.Bucket; want != got {
		t.Fatalf("expected S3 get to bucket %q, got %q", want, got)
	}

	if want, got := prefix+"/some-key", *s3c.get.input.Key; want != got {
		t.Fatalf("expected S3 get to key %q, got %q", want, got)
	}
}

type fakeS3 struct {
	put struct {
		input *s3.PutObjectInput
		body  string
	}

	get struct {
		body  string
		input *s3.GetObjectInput
	}
}

func (f *fakeS3) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	f.put.input = input

	b, err := ioutil.ReadAll(input.Body)
	if err != nil {
		panic("error reading body")
	}
	f.put.body = string(b)
	return nil, nil
}

func (f *fakeS3) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	f.get.input = input
	return &s3.GetObjectOutput{Body: ioutil.NopCloser(strings.NewReader(f.get.body))}, nil
}

type fakeSerializer struct {
	sReq string
	dRes string
}

func (f *fakeSerializer) Serialize(data interface{}) (string, error) {
	f.sReq = data.(string)
	return f.sReq, nil
}

func (f *fakeSerializer) Deserialize(ser string, data interface{}) error {
	v := data.(*string)
	*v = f.dRes
	return nil
}
