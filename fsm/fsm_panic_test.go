package fsm

import (
	"regexp"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/log"
)

type TypeData struct{}

// Note - these vars must remain in the same place, or the expected panic lines
// should be updated
func untypedDecider(f *FSMContext, e *swf.HistoryEvent, data interface{}) Outcome {
	panic("can you handle it?")
}

func typedDeciderFunc(f *FSMContext, h *swf.HistoryEvent, d *TypeData) Outcome {
	panic("lol i tricked you")
}

var typedDecider = Typed(new(TypeData)).Decider(typedDeciderFunc)

var typedDeciderAnon = Typed(new(TypeData)).Decider(func(f *FSMContext, h *swf.HistoryEvent, d *TypeData) Outcome {
	panic("lol i tricked you")
})

var fileMatch = regexp.MustCompile(`file=\"([\w\/\.]+):(\d+)\"`)
var fnMatch = regexp.MustCompile(`func=\"([\w\/\.]+)\"`)

func TestPanicRecovery(t *testing.T) {
	for _, tc := range []struct {
		name    string
		decider Decider
		data    interface{}
		fn      string
		file    string
		line    string
	}{
		{
			name:    "Non-typed",
			decider: untypedDecider,
			data:    nil,
			fn:      "fsm.untypedDecider",
			file:    "fsm_panic_test.go",
			line:    "17", // this is the func above
		},
		{
			name:    "Typed",
			decider: typedDecider,
			data:    &TypeData{},
			fn:      "fsm.typedDeciderFunc",
			file:    "fsm_panic_test.go",
			line:    "21", // this is the func above
		},
		{
			name:    "Typed Anonymous Function",
			decider: typedDeciderAnon,
			data:    &TypeData{},
			fn:      "fsm.glob..func",
			file:    "fsm_panic_test.go",
			line:    "27", // this is the func above
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cl := &log.CapturingLogger{}
			ts := &FSMState{
				Name:    "panic",
				Decider: tc.decider,
			}
			tf := &FSM{Logger: cl}
			tf.AddInitialState(ts)
			_, err := tf.panicSafeDecide(ts, new(FSMContext), &swf.HistoryEvent{}, tc.data)
			if err == nil {
				t.Errorf("%s: Panic expected, but not received", tc.name)
			} else {
				t.Log(err)
			}
			var fn, file, line string
			for _, l := range cl.Lines {
				if m := fileMatch.FindStringSubmatch(l); m != nil {
					file = m[1]
					line = m[2]
				}
				if m := fnMatch.FindStringSubmatch(l); m != nil {
					fn = m[1]
				}
			}
			if !strings.Contains(fn, tc.fn) || !strings.Contains(file, tc.file) || line != tc.line {
				t.Errorf("Expected data not found in log line, found file: %s, line: %s, func: %s, expected file: %s, line: %s, func: %s",
					file, line, fn, tc.file, tc.line, tc.fn)
			}
		})
	}
}
