package test

import (
	"encoding/json"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/endpoints"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"log"
	"net/http"
	"net/http/httptest"
)

type SWFService struct {
	Server *httptest.Server
	Client *swf.SWF
}

func NewSWFService() *SWFService {
	region := "test-region"
	service := new(SWFService)
	server := httptest.NewServer(service)
	service.Server = server
	endpoints.AddOverride("SWF", region, server.URL)
	client := swf.New(aws.Creds("key", "secret", "token"), "test-region", nil)
	service.Client = client
	return service
}

func (*SWFService) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	operation := req.Header.Get("X-Amz-Target")
	switch operation {
	case "SimpleWorkflowService.ListDomains":
		in := new(swf.ListDomainsInput)
		err := json.NewDecoder(req.Body).Decode(in)
		log.Printf("%+v, %s", in, err)
	default:
		log.Println("NOPE")
	}
}
