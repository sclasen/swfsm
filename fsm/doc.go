/*
Package fsm layers an erlang/akka style finite state machine abstraction on top of SWF, and facilitates modeling your workflows as FSMs. The FSM will be responsible for handling the decision
tasks in your workflow that implicitly model it.

The FSM takes care of serializing/deserializing and threading a data model through the workflow history for you, as well as serialization/deserialization of any payloads in events your workflows recieve,
as well as optionally sending the data model snapshots to kinesis, to facilitate a CQRS style application where the query models will be built off the Kinesis stream.

From http://www.erlang.org/doc/design_principles/fsm.html, a finite state machine, or FSM, can be described as a set of relations of the form:

    State(S) x Event(E) -> Actions(A), State(S')

Substituting the relevant SWF/swf4go concepts, we get

   (Your main data struct) x (an swf.HistoryEvent) -> (zero or more swf.Decisions), (A possibly updated main data struct)

See the http://godoc.org/github.com/sclasen/swfsm/fsm#example-FSM for a complete usage example.

*/
package fsm
