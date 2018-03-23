swfsm
======

* simple workflow finite state machines

go library that provides a Finite State Machine abstraction and other niceties on top of swf apis.

 Built using the [aws-sdk-go](https://github.com/awslabs/aws-sdk-go)

[![Build Status](https://travis-ci.org/sclasen/swfsm.svg?branch=master)](https://travis-ci.org/sclasen/swfsm)

* activity godoc here: http://godoc.org/github.com/sclasen/swfsm/activity
* fsm godoc here: http://godoc.org/github.com/sclasen/swfsm/fsm
* poller godoc here: http://godoc.org/github.com/sclasen/swfsm/poller
* migrator godoc here: http://godoc.org/github.com/sclasen/swfsm/migrator
* sugar godoc here: http://godoc.org/github.com/sclasen/swfsm/sugar


features
--------

* Pollers for both ActivityTasks and DecisionTasks which work around some of the eccentricities of the swf service.

* erlang/akka style Finite State Machine abstraction, which is used to model workflows as FSMs.

* primitives for composing the event processing logic for each state in your FSMs.

* migrators that make sure expected Domains, WorkflowTypes, ActivityTypes, KinesisStreams and DynamoDB tables are created.

Please see the godoc for detailed documentation and examples.

versions
--------

Please see vendor/vendor.json for the version of aws-sdk-go that this lib currently supports.
