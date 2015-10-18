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
	GetError() error
}

type historySegmentor struct {
	c              *client
	sink           func(HistorySegment)
	segment        HistorySegment
	err            error
	refs           map[int64][]*int64
	nextCorrelator *EventCorrelator
	nextErrorState *SerializedErrorState
}

func newHistorySegmentor(c *client, sink func(HistorySegment)) *historySegmentor {
	return &historySegmentor{
		c:       c,
		sink:    sink,
		segment: HistorySegment{Events: []*HistorySegmentEvent{}},
		refs:    make(map[int64][]*int64),
	}
}

// must call s.GetError after completion to check no errors exist
func (s *historySegmentor) FromPage(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool) {
	for _, event := range p.Events {
		if s.err = s.process(event); s.err != nil {
			return false
		}
	}

	if lastPage {
		s.flush()
	}

	return true
}

func (s *historySegmentor) GetError() error {
	return s.err
}

func (s *historySegmentor) process(event *swf.HistoryEvent) error {
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
			s.sink(s.segment)
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

func (s *historySegmentor) flush() {
	if s.segment.State != nil {
		s.sink(s.segment)
	}
}
