package migrator

import (
	"log"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/kinesis"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
	//"github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"fmt"
	"time"
)

// TypesMigrator is composed of a DomainMigrator, a WorkflowTypeMigrator and an ActivityTypeMigrator.
type TypesMigrator struct {
	DomainMigrator       *DomainMigrator
	WorkflowTypeMigrator *WorkflowTypeMigrator
	ActivityTypeMigrator *ActivityTypeMigrator
	StreamMigrator       *StreamMigrator
}

type SWFOps interface {
	DeprecateActivityType(req *swf.DeprecateActivityTypeInput) (err error)
	DeprecateDomain(req *swf.DeprecateDomainInput) (err error)
	DeprecateWorkflowType(req *swf.DeprecateWorkflowTypeInput) (err error)
	DescribeActivityType(req *swf.DescribeActivityTypeInput) (resp *swf.ActivityTypeDetail, err error)
	DescribeDomain(req *swf.DescribeDomainInput) (resp *swf.DomainDetail, err error)
	DescribeWorkflowExecution(req *swf.DescribeWorkflowExecutionInput) (resp *swf.WorkflowExecutionDetail, err error)
	DescribeWorkflowType(req *swf.DescribeWorkflowTypeInput) (resp *swf.WorkflowTypeDetail, err error)
	RegisterActivityType(req *swf.RegisterActivityTypeInput) (err error)
	RegisterDomain(req *swf.RegisterDomainInput) (err error)
	RegisterWorkflowType(req *swf.RegisterWorkflowTypeInput) (err error)
}

type KinesisOps interface {
	CreateStream(req *kinesis.CreateStreamInput) (err error)
	DescribeStream(req *kinesis.DescribeStreamInput) (resp *kinesis.DescribeStreamOutput, err error)
}

// Migrate runs Migrate on the underlying DomainMigrator, a WorkflowTypeMigrator and ActivityTypeMigrator.
func (t *TypesMigrator) Migrate() {
	if t.ActivityTypeMigrator == nil {
		t.ActivityTypeMigrator = new(ActivityTypeMigrator)
	}
	if t.DomainMigrator == nil {
		t.DomainMigrator = new(DomainMigrator)
	}
	if t.WorkflowTypeMigrator == nil {
		t.WorkflowTypeMigrator = new(WorkflowTypeMigrator)
	}
	if t.StreamMigrator == nil {
		t.StreamMigrator = new(StreamMigrator)
	}

	t.DomainMigrator.Migrate()
	ParallelMigrate(
		t.WorkflowTypeMigrator,
		t.ActivityTypeMigrator,
		t.StreamMigrator,
	)
}

type Migration interface {
	Migrate()
}

func ParallelMigrate(migrators ...Migration) {
	fail := make(chan interface{})
	done := make(chan struct{}, len(migrators))
	for _, m := range migrators {
		migrator := m //capture ref for goroutime
		go func() {
			defer func() {
				if r := recover(); r != nil {
					fail <- r
				}
			}()
			migrator.Migrate()
			done <- struct{}{}
		}()
	}
	for range migrators {
		select {
		case <-done:
		case e := <-fail:
			log.Panicf("migrator failed: %v", e)
		}
	}
}

// DomainMigrator will register or deprecate the configured domains as required.
type DomainMigrator struct {
	RegisteredDomains []swf.RegisterDomainInput
	DeprecatedDomains []swf.DeprecateDomainInput
	Client            SWFOps
}

// Migrate asserts that DeprecatedDomains are deprecated or deprecates them, then asserts that RegisteredDomains are registered or registers them.
func (d *DomainMigrator) Migrate() { //add parallel migrations to all Migrate!
	for _, dd := range d.DeprecatedDomains {
		if d.isDeprecated(dd.Name) {
			log.Printf("action=migrate at=deprecate-domain domain=%s status=previously-deprecated", LS(dd.Name))
		} else {
			d.deprecate(dd)
			log.Printf("action=migrate at=deprecate-domain domain=%s status=deprecated", LS(dd.Name))
		}
	}
	for _, r := range d.RegisteredDomains {
		if d.isRegisteredNotDeprecated(r) {
			log.Printf("action=migrate at=register-domain domain=%s status=previously-registered", LS(r.Name))
		} else {
			d.register(r)
			log.Printf("action=migrate at=register-domain domain=%s status=registered", LS(r.Name))
		}
	}
}

func (d *DomainMigrator) isRegisteredNotDeprecated(rd swf.RegisterDomainInput) bool {
	desc, err := d.describe(rd.Name)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeUnknownResourceFault {
			return false
		}

		panicWithError(err)

	}

	return *desc.DomainInfo.Status == swf.RegistrationStatusRegistered
}

func (d *DomainMigrator) register(rd swf.RegisterDomainInput) {
	err := d.Client.RegisterDomain(&rd)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeDomainAlreadyExistsFault {
			return
		}

		panicWithError(err)

	}
}

func (d *DomainMigrator) isDeprecated(domain aws.StringValue) bool {
	desc, err := d.describe(domain)
	if err != nil {
		log.Printf("action=migrate at=is-dep domain=%s error=%s", LS(domain), err.Error())
		return false
	}

	return *desc.DomainInfo.Status == swf.RegistrationStatusDeprecated
}

func (d *DomainMigrator) deprecate(dd swf.DeprecateDomainInput) {
	err := d.Client.DeprecateDomain(&dd)
	if err != nil {
		panicWithError(err)
	}
}

func (d *DomainMigrator) describe(domain aws.StringValue) (*swf.DomainDetail, error) {
	resp, err := d.Client.DescribeDomain(&swf.DescribeDomainInput{Name: domain})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// WorkflowTypeMigrator will register or deprecate the configured workflow types as required.
type WorkflowTypeMigrator struct {
	RegisteredWorkflowTypes []swf.RegisterWorkflowTypeInput
	DeprecatedWorkflowTypes []swf.DeprecateWorkflowTypeInput
	Client                  SWFOps
}

// Migrate asserts that DeprecatedWorkflowTypes are deprecated or deprecates them, then asserts that RegisteredWorkflowTypes are registered or registers them.
func (w *WorkflowTypeMigrator) Migrate() {
	for _, dd := range w.DeprecatedWorkflowTypes {
		if w.isDeprecated(dd.Domain, dd.WorkflowType.Name, dd.WorkflowType.Version) {
			log.Printf("action=migrate at=deprecate-workflow domain=%s workflow=%s version=%s status=previously-deprecated", LS(dd.Domain), LS(dd.WorkflowType.Name), LS(dd.WorkflowType.Version))
		} else {
			w.deprecate(dd)
			log.Printf("action=migrate at=deprecate-workflow domain=%s  workflow=%s version=%s status=deprecate", LS(dd.Domain), LS(dd.WorkflowType.Name), LS(dd.WorkflowType.Version))
		}
	}
	for _, r := range w.RegisteredWorkflowTypes {
		if w.isRegisteredNotDeprecated(r) {
			log.Printf("action=migrate at=register-workflow domain=%s workflow=%s version=%s status=previously-registered", LS(r.Domain), LS(r.Name), LS(r.Version))
		} else {
			w.register(r)
			log.Printf("action=migrate at=register-workflow domain=%s  workflow=%s version=%s status=registered", LS(r.Domain), LS(r.Name), LS(r.Version))
		}
	}
}

func (w *WorkflowTypeMigrator) isRegisteredNotDeprecated(rd swf.RegisterWorkflowTypeInput) bool {
	desc, err := w.describe(rd.Domain, rd.Name, rd.Version)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeUnknownResourceFault {
			return false
		}

		panicWithError(err)

	}

	return *desc.TypeInfo.Status == swf.RegistrationStatusRegistered
}

func (w *WorkflowTypeMigrator) register(rd swf.RegisterWorkflowTypeInput) {
	err := w.Client.RegisterWorkflowType(&rd)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeAlreadyExistsFault {
			return
		}

		panicWithError(err)
	}
}

func (w *WorkflowTypeMigrator) isDeprecated(domain aws.StringValue, name aws.StringValue, version aws.StringValue) bool {
	desc, err := w.describe(domain, name, version)
	if err != nil {
		log.Printf("action=migrate at=is-dep domain=%s workflow=%s version=%s error=%s", LS(domain), LS(name), LS(version), err.Error())
		return false
	}

	return *desc.TypeInfo.Status == swf.RegistrationStatusDeprecated
}

func (w *WorkflowTypeMigrator) deprecate(dd swf.DeprecateWorkflowTypeInput) {
	err := w.Client.DeprecateWorkflowType(&dd)
	if err != nil {
		panicWithError(err)
	}
}

func (w *WorkflowTypeMigrator) describe(domain aws.StringValue, name aws.StringValue, version aws.StringValue) (*swf.WorkflowTypeDetail, error) {
	resp, err := w.Client.DescribeWorkflowType(&swf.DescribeWorkflowTypeInput{Domain: domain, WorkflowType: &swf.WorkflowType{Name: name, Version: version}})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ActivityTypeMigrator will register or deprecate the configured activity types as required.
type ActivityTypeMigrator struct {
	RegisteredActivityTypes []swf.RegisterActivityTypeInput
	DeprecatedActivityTypes []swf.DeprecateActivityTypeInput
	Client                  SWFOps
}

// Migrate asserts that DeprecatedActivityTypes are deprecated or deprecates them, then asserts that RegisteredActivityTypes are registered or registers them.
func (a *ActivityTypeMigrator) Migrate() {
	for _, d := range a.DeprecatedActivityTypes {
		if a.isDeprecated(d.Domain, d.ActivityType.Name, d.ActivityType.Version) {
			log.Printf("action=migrate at=deprecate-activity domain=%s activity=%s version=%s status=previously-deprecated", LS(d.Domain), LS(d.ActivityType.Name), LS(d.ActivityType.Version))
		} else {
			a.deprecate(d)
			log.Printf("action=migrate at=depreacate-activity domain=%s activity=%s version=%s status=deprecated", LS(d.Domain), LS(d.ActivityType.Name), LS(d.ActivityType.Version))
		}
	}
	for _, r := range a.RegisteredActivityTypes {
		if a.isRegisteredNotDeprecated(r) {
			log.Printf("action=migrate at=register-activity domain=%s activity=%s version=%s status=previously-registered", LS(r.Domain), LS(r.Name), LS(r.Version))
		} else {
			a.register(r)
			log.Printf("action=migrate at=register-activity domain=%s activity=%s version=%s status=registered", LS(r.Domain), LS(r.Name), LS(r.Version))
		}
	}
}

func (a *ActivityTypeMigrator) isRegisteredNotDeprecated(rd swf.RegisterActivityTypeInput) bool {
	desc, err := a.describe(rd.Domain, rd.Name, rd.Version)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeUnknownResourceFault {
			return false
		}

		panicWithError(err)

	}

	return *desc.TypeInfo.Status == swf.RegistrationStatusRegistered
}

func (a *ActivityTypeMigrator) register(rd swf.RegisterActivityTypeInput) {
	err := a.Client.RegisterActivityType(&rd)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeAlreadyExistsFault {
			return
		}

		panicWithError(err)
	}
}

func (a *ActivityTypeMigrator) isDeprecated(domain aws.StringValue, name aws.StringValue, version aws.StringValue) bool {
	desc, err := a.describe(domain, name, version)
	if err != nil {
		log.Printf("action=migrate at=is-dep domain=%s activity=%s version=%s error=%s", LS(domain), LS(name), LS(version), err.Error())
		return false
	}

	return *desc.TypeInfo.Status == swf.RegistrationStatusDeprecated
}

func (a *ActivityTypeMigrator) deprecate(dd swf.DeprecateActivityTypeInput) {
	err := a.Client.DeprecateActivityType(&dd)
	if err != nil {
		panicWithError(err)
	}
}

func (a *ActivityTypeMigrator) describe(domain aws.StringValue, name aws.StringValue, version aws.StringValue) (*swf.ActivityTypeDetail, error) {
	resp, err := a.Client.DescribeActivityType(&swf.DescribeActivityTypeInput{Domain: domain, ActivityType: &swf.ActivityType{Name: name, Version: version}})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// StreamMigrator will create any Kinesis Streams required.
type StreamMigrator struct {
	Streams []kinesis.CreateStreamInput
	Client  KinesisOps
	Timeout int
}

// Migrate checks that the desired streams have been created and if they have not, creates them.s
func (s *StreamMigrator) Migrate() {
	for _, st := range s.Streams {
		if s.isCreated(st) {
			log.Printf("action=migrate at=create-stream stream=%s status=previously-created", LS(st.StreamName))
		} else {
			s.create(st)
			log.Printf("action=migrate at=create-stream stream=%s status=created", LS(st.StreamName))
		}
		s.awaitActive(st.StreamName, s.Timeout)
	}
}

func (s *StreamMigrator) isCreated(st kinesis.CreateStreamInput) bool {
	_, err := s.describe(st)
	if err != nil {
		if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeStreamNotFound {
			return false
		}
		panicWithError(err)

	}

	return true
}

func (s *StreamMigrator) create(st kinesis.CreateStreamInput) {
	err := s.Client.CreateStream(&st)
	if ae, ok := err.(aws.APIError); ok && ae.Type == ErrorTypeStreamAlreadyExists {
		return
	}
}

func (s *StreamMigrator) describe(st kinesis.CreateStreamInput) (*kinesis.DescribeStreamOutput, error) {
	req := kinesis.DescribeStreamInput{
		StreamName: st.StreamName,
	}
	resp, err := s.Client.DescribeStream(&req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *StreamMigrator) awaitActive(stream aws.StringValue, atMostSeconds int) {

	waited := 0
	status := kinesis.StreamStatusCreating
	for status != kinesis.StreamStatusActive {
		desc, err := s.Client.DescribeStream(&kinesis.DescribeStreamInput{
			StreamName: stream,
		})
		if err != nil {
			log.Printf("component=kinesis-migrator fn=awaitActive at=describe-error error=%s", err)
			panicWithError(err)
		}
		log.Printf("component=kinesis-migrator fn=awaitActive stream=%s at=describe status=%s", *stream, *desc.StreamDescription.StreamStatus)
		status = *desc.StreamDescription.StreamStatus
		time.Sleep(1 * time.Second)
		waited++
		if waited >= atMostSeconds {
			log.Printf("component=kinesis-migrator fn=awaitActive streal=%s  at=error error=exeeeded-max-wait", *stream)
			panic("waited too long")
		}
	}
}

func panicWithError(err error) {
	if ae, ok := err.(aws.APIError); ok {
		panic(fmt.Sprintf("aws error while migrating type=%s message=%s code=%s request-id=%s", ae.Type, ae.Message, ae.Code, ae.RequestID))
	}

	panic(err)
}
