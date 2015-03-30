swfsm
======

* simple workflow finite state machines

go library that provides a Finite State Machine abstraction and other niceities on top of swf apis.

 Built using the [aws-sdk-go](https://github.com/awslabs/aws-sdk-go)

[![Build Status](https://travis-ci.org/sclasen/swfsm.svg?branch=master)](https://travis-ci.org/sclasen/swfsm)

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

Since this is a library, and using Godeps in your library is discouraged by the Godeps folks, it is important to note here
that the version of aws-sdk-go was build against and tested with is `295e08b503c4ee9fbcdc9aa1aa06966a16364e5a`

There have been subsequent breaking changes merged to master in that library, which we have opened tickets to get fixed. (All the consts were, perhaps inadvertently, removed)
