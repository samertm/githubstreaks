package debug

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/samertm/githubstreaks/conf"
	"github.com/segmentio/go-loggly"
	"github.com/tj/go-debug"
)

var Logf = debug.Debug("githubstreaks")

func init() {
	if conf.Config.Debug != "" {
		debug.Enable(conf.Config.Debug)
	}
	var w io.Writer
	if conf.Config.LogglyToken != "" {
		log.Println("Logging to Loggly")
		logglyClient = loggly.New(conf.Config.LogglyToken, "githubstreaks")
		logglyClient.Writer = os.Stderr
		w = logglyClient
	} else {
		log.Println("Logging to stderr.")
		w = os.Stderr
	}
	Logger = log.New(w, "", log.LstdFlags|log.Lshortfile)
}

var logglyClient *loggly.Client

// flush flushes the logger. No-op if we are not logging to Loggly.
// Must be called before the program exits abruptly while logging.
func flush() {
	if logglyClient != nil {
		logglyClient.Flush()
	}
}

var Logger *log.Logger

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func Fatal(v ...interface{}) {
	Print(v...)
	flush()
	os.Exit(1)
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func Fatalf(format string, v ...interface{}) {
	Printf(format, v)
	flush()
	os.Exit(1)
}

// Fatalln is equivalent to Println() followed by a call to
// os.Exit(1).
func Fatalln(v ...interface{}) {
	Println(v...)
	flush()
	os.Exit(1)
}

// Panic is equivalent to Print() followed by a call to panic().
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	Print(v...)
	flush()
	panic(s)
}

// Panicf is equivalent to Printf() followed by a call to panic().
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	Printf(format, v...)
	flush()
	panic(s)
}

// Panicln is equivalent to Println() followed by a call to panic().
func Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	Println(v...)
	flush()
	panic(s)
}

// Print calls l.Output to print to the logger.
// Arguments are handled in the manner of fmt.Print.
func Print(v ...interface{}) {
	Logger.Print(v...)
}

// Printf calls Output to print to the standard logger. Arguments are
// handled in the manner of fmt.Printf.
func Printf(format string, v ...interface{}) {
	Logger.Printf(format, v...)
}

// Println calls Output to print to the standard logger. Arguments are
// handled in the manner of fmt.Println.
func Println(v ...interface{}) {
	Logger.Println(v...)
}
