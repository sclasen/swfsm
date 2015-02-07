package test

import (
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"testing"
)

func TestSWFServer(t *testing.T) {
	service := NewSWFService()
	service.Client.ListDomains(&swf.ListDomainsInput{RegistrationStatus: aws.String(swf.RegistrationStatusRegistered)})
}
