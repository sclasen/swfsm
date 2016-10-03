package log

import (
	"fmt"
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

// CapturingLogger is designed to be used in testing - it will saves lines it receives
type CapturingLogger struct {
	Lines []string
}

func (c *CapturingLogger) Print(a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprint(a...))
}
func (c *CapturingLogger) Printf(s string, a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprintf(s, a...))
}
func (c *CapturingLogger) Println(a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprint(a...))
}
func (c *CapturingLogger) Fatal(a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprint(a...))
}
func (c *CapturingLogger) Fatalf(s string, a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprintf(s, a...))
}
func (c *CapturingLogger) Fatalln(a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprint(a...))
}
func (c *CapturingLogger) Panic(a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprint(a...))
}
func (c *CapturingLogger) Panicf(s string, a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprintf(s, a...))
}
func (c *CapturingLogger) Panicln(a ...interface{}) {
	c.Lines = append(c.Lines, fmt.Sprint(a...))
}
