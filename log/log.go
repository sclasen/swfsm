package log

import (
	golog "log"
	"os"
)

//provide a mutable logger so it can be changed
//this is what the default logger in go's log pakcage looks like
var Log = golog.New(os.Stderr, "", golog.LstdFlags)
