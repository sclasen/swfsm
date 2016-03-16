package s3serializer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pborman/uuid"
	swfsm "github.com/sclasen/swfsm/fsm"
	. "github.com/sclasen/swfsm/log"
)

const (
	urlScheme   = "s3+ser"
	magicPrefix = urlScheme + "://"
)

var (
	// if underlying serializer returns data less than this
	// we use its result directly without involving S3
	defaultMinS3Length = 32000
	minS3Length        = defaultMinS3Length

	defaultKeyGen = func() string { return uuid.New() }
	keyGen        = defaultKeyGen
)

type S3Ops interface {
	PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

type S3Serializer struct {
	s3c      S3Ops
	s3Bucket string
	s3Prefix string

	under swfsm.StateSerializer
}

func New(s3c S3Ops, bucket, prefix string, under swfsm.StateSerializer) *S3Serializer {
	return &S3Serializer{s3c, bucket, prefix, under}
}

func (s *S3Serializer) Serialize(state interface{}) (string, error) {
	ser, err := s.under.Serialize(state)
	if err != nil {
		return "", err
	}

	slen := len(ser)
	if slen < minS3Length {
		return ser, nil
	}

	key := keyGen()
	if s.s3Prefix != "" {
		key = s.s3Prefix + "/" + key
	}

	_, err = s.s3c.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.s3Bucket),
		Key:    aws.String(key),
		Body:   strings.NewReader(ser),
	})
	if err != nil {
		Log.Printf("component=s3serializer bucket=%q key=%q slen=%d at=put-error error=%q", s.s3Bucket, key, slen, err)
		return "", err
	}
	Log.Printf("component=s3serializer bucket=%q key=%q slen=%d at=put-success", s.s3Bucket, key, slen)

	u := &url.URL{}
	u.Scheme = urlScheme
	u.Host = s.s3Bucket
	u.Path = key

	return u.String(), nil
}

func (s *S3Serializer) Deserialize(deser string, state interface{}) error {
	if !strings.HasPrefix(deser, magicPrefix) {
		return s.under.Deserialize(deser, state)
	}

	u, err := url.Parse(deser)
	if err != nil {
		return fmt.Errorf("problem parsing s3 object URL: " + err.Error())
	}

	// TODO: warn/error on unexpected bucket?
	// TODO: warn/error on too-short path and/or unexpected prefix?

	if len(u.Path) < 2 { // '/' + at least one char
		// any worry about leaking sensitive things here?
		// at some point there could be creds
		return errors.New("s3 object URL path too short")
	}
	key := u.Path[1:]

	resp, err := s.s3c.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(key),
	})
	if err != nil {
		Log.Printf("component=s3serializer bucket=%q key=%q at=get-error error=%q", s.s3Bucket, key, err)
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.Printf("component=s3serializer bucket=%q key=%q at=get-error error=%q", s.s3Bucket, key, err)
		return err
	}
	Log.Printf("component=s3serializer bucket=%q key=%q slen=%d at=get-success", s.s3Bucket, key, len(b))

	return s.under.Deserialize(string(b), state)
}
