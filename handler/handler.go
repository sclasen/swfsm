package handler

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	. "github.com/sclasen/swfsm/log"
)

//copy of aws.SendHandler modified to use different http clients for timeouts
//on polling and heartbeating.
//to use, when constructing an swf.SWM
// swfClient.Service.Handlers.Send.Clear()
// swfClient.Service.Handlers.Send.PushBack(handler.SWFSendHandler(polling, heartbeat))
func SWFSendHandler(polling, heartbeat *http.Client) func(*request.Request) {
	var reStatusCode = regexp.MustCompile(`^(\d+)`)

	return func(r *request.Request) {
		client := r.Config.HTTPClient
		if r.ClientInfo.ServiceName == "swf" {
			switch r.Operation.Name {
			case "PollForDecisionTask", "PollForActivityTask":
				if r.Config.LogLevel.AtLeast(aws.LogDebug) {
					Log.Printf("using polling client %s %s", r.ClientInfo.ServiceName, r.Operation.Name)
				}
				client = polling
			case "RecordActivityTaskHeartbeat":
				if r.Config.LogLevel.AtLeast(aws.LogDebug) {
					Log.Printf("using heartbeat client %s %s", r.ClientInfo.ServiceName, r.Operation.Name)
				}
				client = heartbeat
			default:
				if r.Config.LogLevel.AtLeast(aws.LogDebug) {
					Log.Printf("using std client %s %s", r.ClientInfo.ServiceName, r.Operation.Name)
				}
			}

		}

		var err error
		r.HTTPResponse, err = client.Do(r.HTTPRequest)
		if err != nil {
			// Prevent leaking if an HTTPResponse was returned. Clean up
			// the body.
			if r.HTTPResponse != nil {
				r.HTTPResponse.Body.Close()
			}
			// Capture the case where url.Error is returned for error processing
			// response. e.g. 301 without location header comes back as string
			// error and r.HTTPResponse is nil. Other url redirect errors will
			// comeback in a similar method.
			if e, ok := err.(*url.Error); ok && e.Err != nil {
				if s := reStatusCode.FindStringSubmatch(e.Err.Error()); s != nil {
					code, _ := strconv.ParseInt(s[1], 10, 64)
					r.HTTPResponse = &http.Response{
						StatusCode: int(code),
						Status:     http.StatusText(int(code)),
						Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
					}
					return
				}
			}
			if r.HTTPResponse == nil {
				// Add a dummy request response object to ensure the HTTPResponse
				// value is consistent.
				r.HTTPResponse = &http.Response{
					StatusCode: int(0),
					Status:     http.StatusText(int(0)),
					Body:       ioutil.NopCloser(bytes.NewReader([]byte{})),
				}
			}
			// Catch all other request errors.
			r.Error = awserr.New("RequestError", "send request failed", err)
			r.Retryable = aws.Bool(true) // network errors are retryable
		}
	}
}
