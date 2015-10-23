package fsm

import (
	"fmt"

	"strings"

	"encoding/json"
	"reflect"

	"strconv"

	"time"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

type HistorySegment struct {
	State                   *HistorySegmentState
	Correlator              *EventCorrelator
	Error                   *SerializedErrorState
	Events                  []*HistorySegmentEvent
	ContinuedExecutionRunId *string
}

type HistorySegmentState struct {
	ID        *int64
	Timestamp *time.Time
	Version   *uint64
	Name      *string
	Data      *interface{}
}

type HistorySegmentEvent struct {
	ID         *int64
	Timestamp  *time.Time
	Type       *string
	Attributes *map[string]interface{}
	References []*int64
}

type HistorySegmentor interface {
	FromPage(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool)
	OnStart(fn func()) HistorySegmentor
	OnSegment(func(HistorySegment)) HistorySegmentor
	OnPage(fn func()) HistorySegmentor
	OnError(func(error)) HistorySegmentor
	OnFinish(fn func()) HistorySegmentor
}

type historySegmentor struct {
	c         *client
	onStart   func()
	onSegment func(HistorySegment)
	onPage    func()
	onError   func(error)
	onFinish  func()

	started        bool
	finished       bool
	segment        HistorySegment
	refs           map[int64][]*int64
	nextCorrelator *EventCorrelator
	nextErrorState *SerializedErrorState
}

func NewHistorySegmentor(c *client) *historySegmentor {
	return &historySegmentor{
		c:         c,
		onStart:   func() {},
		onSegment: func(_ HistorySegment) {},
		onPage:    func() {},
		onError:   func(_ error) {},
		onFinish:  func() {},
		segment:   HistorySegment{Events: []*HistorySegmentEvent{}},
		refs:      make(map[int64][]*int64),
	}
}

func (s *historySegmentor) OnStart(fn func()) HistorySegmentor {
	s.onStart = fn
	return s
}

func (s *historySegmentor) OnSegment(fn func(HistorySegment)) HistorySegmentor {
	s.onSegment = fn
	return s
}

func (s *historySegmentor) OnPage(fn func()) HistorySegmentor {
	s.onPage = fn
	return s
}

func (s *historySegmentor) OnError(fn func(error)) HistorySegmentor {
	s.onError = fn
	return s
}

func (s *historySegmentor) OnFinish(fn func()) HistorySegmentor {
	s.onFinish = fn
	return s
}

func (s *historySegmentor) FromPage(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool) {
	for _, event := range p.Events {
		if err := s.process(event); err != nil {
			s.onError(err)
			return false
		}
	}

	if lastPage {
		s.finish()
	}

	s.onPage()
	return true
}

func (s *historySegmentor) process(event *swf.HistoryEvent) error {
	if s.finished {
		return fmt.Errorf("Cannot process more events after finising segmentor. Create a new segmentor.")
	}

	if !s.started {
		s.started = true
		s.onStart()
	}

	unrecordedName := "<unrecorded>"
	unrecordedId := int64(999999)
	unrecordedVersion := uint64(999999)

	if s.c.f.isCorrelatorMarker(event) {
		correlator, err := s.c.f.findSerializedEventCorrelator([]*swf.HistoryEvent{event})
		if err != nil {
			return err
		}
		s.nextCorrelator = correlator
		return nil
	}

	if s.c.f.isErrorMarker(event) {
		errorState, err := s.c.f.findSerializedErrorState([]*swf.HistoryEvent{event})
		if err != nil {
			return err
		}
		s.nextErrorState = errorState
		return nil
	}

	state, err := s.c.f.statefulHistoryEventToSerializedState(event)
	if err != nil {
		return err
	}

	if state != nil {
		if s.segment.State != nil {
			s.onSegment(s.segment)
			s.segment = HistorySegment{Events: []*HistorySegmentEvent{}}
		}

		data := s.c.f.zeroStateData()
		s.segment.State = &HistorySegmentState{
			ID:        event.EventId,
			Timestamp: event.EventTimestamp,
			Version:   &state.StateVersion,
			Name:      S(state.StateName),
			Data:      &data,
		}
		err = s.c.f.Serializer.Deserialize(state.StateData, s.segment.State.Data)
		if err != nil {
			return err
		}

		if event.WorkflowExecutionStartedEventAttributes != nil {
			s.segment.ContinuedExecutionRunId = event.WorkflowExecutionStartedEventAttributes.ContinuedExecutionRunId
		}

		s.segment.Correlator = s.nextCorrelator
		s.nextCorrelator = nil

		s.segment.Error = s.nextErrorState
		s.nextErrorState = nil

		return nil
	}

	if s.segment.State == nil {
		s.segment.State = &HistorySegmentState{
			Name:    &unrecordedName,
			ID:      &(unrecordedId),
			Version: &unrecordedVersion,
		}
	}

	eventAttributes, err := s.transformHistoryEventAttributes(event)
	if err != nil {
		return err
	}

	for key, value := range eventAttributes {
		if strings.HasSuffix(key, "EventId") {
			parsed, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
			if err != nil {
				return err
			}
			s.refs[parsed] = append(s.refs[parsed], event.EventId)
		}
	}

	s.segment.Events = append(s.segment.Events, &HistorySegmentEvent{
		Type:       event.EventType,
		ID:         event.EventId,
		Timestamp:  event.EventTimestamp,
		Attributes: &eventAttributes,
		References: s.refs[*event.EventId],
	})

	return nil
}

func (s *historySegmentor) transformHistoryEventAttributes(e *swf.HistoryEvent) (map[string]interface{}, error) {
	attrStruct := reflect.ValueOf(*e).FieldByName(*e.EventType + "EventAttributes").Interface()
	attrJsonBytes, err := json.Marshal(attrStruct)
	if err != nil {
		return nil, err
	}

	attrMap := make(map[string]interface{})
	err = json.Unmarshal(attrJsonBytes, &attrMap)
	if err != nil {
		return nil, err
	}

	for k, v := range attrMap {
		tryValueMap := make(map[string]interface{})
		tryErr := json.Unmarshal([]byte(fmt.Sprint(v)), &tryValueMap)
		if tryErr == nil {
			attrMap[k] = tryValueMap
		}
	}
	return attrMap, nil
}

func (s *historySegmentor) finish() {
	if s.segment.State != nil {
		s.onSegment(s.segment)
	}
	s.finished = true
	s.onFinish()
}
