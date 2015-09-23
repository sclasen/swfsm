package fsm

import (
	"fmt"

	"strings"

	"encoding/json"
	"reflect"

	"strconv"

	"io"
	"time"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMSnapshot struct {
	State      *FSMSnapshotState
	Correlator *EventCorrelator
	Events     []*FSMSnapshotEvent
}

type FSMSnapshotState struct {
	ID        *int64
	Timestamp *time.Time
	Version   *uint64
	Name      *string
	Data      *interface{}
}

type FSMSnapshotEvent struct {
	ID         *int64
	Timestamp  *time.Time
	Type       *string
	Attributes *map[string]interface{}
	References []*int64
}

type Snapshotter interface {
	FromWorkflowID(id string) ([]FSMSnapshot, error)
	FromWorkflowExecution(exec *swf.WorkflowExecution) ([]FSMSnapshot, error)
	FromReader(reader io.Reader) ([]FSMSnapshot, error)
	FromHistoryEventIterator(itr HistoryEventIterator) ([]FSMSnapshot, error)
}

type snapshotter struct {
	c *client
}

func newSnapshotter(c *client) Snapshotter {
	return &snapshotter{
		c: c,
	}
}

func (s *snapshotter) FromWorkflowID(id string) ([]FSMSnapshot, error) {
	itr, err := s.c.GetHistoryEventIteratorFromWorkflowID(id)
	if err != nil {
		return nil, err
	}
	return s.FromHistoryEventIterator(itr)
}

func (s *snapshotter) FromWorkflowExecution(exec *swf.WorkflowExecution) ([]FSMSnapshot, error) {
	itr, err := s.c.GetHistoryEventIteratorFromWorkflowExecution(exec)
	if err != nil {
		return nil, err
	}
	return s.FromHistoryEventIterator(itr)
}

func (s *snapshotter) FromReader(reader io.Reader) ([]FSMSnapshot, error) {
	itr, err := s.c.GetHistoryEventIteratorFromReader(reader)
	if err != nil {
		return nil, err
	}
	return s.FromHistoryEventIterator(itr)
}

func (s *snapshotter) FromHistoryEventIterator(itr HistoryEventIterator) ([]FSMSnapshot, error) {
	snapshots := []FSMSnapshot{}
	var err error

	zero := s.c.f.zeroStateData()
	unrecordedName := "<unrecorded>"
	unrecordedID := int64(999999)
	unrecordedVersion := uint64(999999)

	refs := make(map[int64][]*int64)
	snapshot := FSMSnapshot{Events: []*FSMSnapshotEvent{}}
	var nextCorrelator *EventCorrelator
	event, err := itr()
	for ; event != nil; event, err = itr() {
		if err != nil {
			return snapshots, err
		}

		if s.c.f.isCorrelatorMarker(event) {
			correlator, err := s.c.f.findSerializedEventCorrelator([]*swf.HistoryEvent{event})
			if err != nil {
				break
			}
			nextCorrelator = correlator
			continue
		}

		state, err := s.c.f.statefulHistoryEventToSerializedState(event)
		if err != nil {
			break
		}

		if state != nil {
			if snapshot.State != nil {
				snapshots = append(snapshots, snapshot)
				snapshot = FSMSnapshot{Events: []*FSMSnapshotEvent{}}
			}

			snapshot.State = &FSMSnapshotState{
				ID:        event.EventID,
				Timestamp: event.EventTimestamp,
				Version:   &state.StateVersion,
				Name:      S(state.StateName),
				Data:      &zero,
			}
			err = s.c.f.Serializer.Deserialize(state.StateData, snapshot.State.Data)
			if err != nil {
				break
			}

			snapshot.Correlator = nextCorrelator
			nextCorrelator = nil

			continue
		}

		if snapshot.State == nil {
			snapshot.State = &FSMSnapshotState{
				Name:    &unrecordedName,
				ID:      &(unrecordedID),
				Version: &unrecordedVersion,
			}
		}

		eventAttributes, err := s.snapshotEventAttributesMap(event)
		if err != nil {
			break
		}

		for key, value := range eventAttributes {
			if strings.HasSuffix(key, "EventID") {
				parsed, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
				if err != nil {
					break
				}
				refs[parsed] = append(refs[parsed], event.EventID)
			}
		}

		snapshot.Events = append(snapshot.Events, &FSMSnapshotEvent{
			Type:       event.EventType,
			ID:         event.EventID,
			Timestamp:  event.EventTimestamp,
			Attributes: &eventAttributes,
			References: refs[*event.EventID],
		})
	}

	if snapshot.State != nil {
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, err
}

func (s *snapshotter) snapshotEventAttributesMap(e *swf.HistoryEvent) (map[string]interface{}, error) {
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
