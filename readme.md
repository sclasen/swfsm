swf-go
======

go library that provides a Finite State Machine abstraction and other niceities on top of swf apis.

 Built using the [aws-sdk-go](https://github.com/awslabs/aws-sdk-go)

[![Build Status](https://travis-ci.org/sclasen/swf-go.svg?branch=master)](https://travis-ci.org/sclasen/swf-go)

* fsm godoc here: http://godoc.org/github.com/sclasen/swf-go/fsm
* poller godoc here: http://godoc.org/github.com/sclasen/swf-go/poller
* migrator godoc here: http://godoc.org/github.com/sclasen/swf-go/migrator
* sugar godoc here: http://godoc.org/github.com/sclasen/swf-go/sugar


features
--------

* Pollers for both ActivityTasks and DecisionTasks which work around some of the eccentricities of the swf service.

* erlang/akka style Finite State Machine abstraction, which is used to model workflows as FSMs.

* primitives for composing the event processing logic for each state in your FSMs.

* migrators that make sure expected Domains, WorkflowTypes, ActivityTypes, KinesisStreams and DynamoDB tables are created.

Please see the godoc for detailed documentation and examples.
