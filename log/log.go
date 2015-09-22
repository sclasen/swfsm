package log

import (
	golog "log"
	"os"
)

// Won't compile if StdLogger can't be realized by a log.Logger
var _ StdLogger = &golog.Logger{}

// StdLogger is what your logrus-enabled library should take, that way
// it'll accept a stdlib logger and a logrus logger. There's no standard
// interface, this is the closest we get, unfortunately.
type StdLogger interface {
	Print(...interface{})
	Printf(string, ...interface{})
	Println(...interface{})

	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Fatalln(...interface{})

	Panic(...interface{})
	Panicf(string, ...interface{})
	Panicln(...interface{})
}

//provide a mutable logger so it can be changed
//this is what the default logger in go's log pakcage looks like
var Log StdLogger = golog.New(os.Stderr, "", golog.LstdFlags)
