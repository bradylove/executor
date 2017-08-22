package log_streamer

import (
	"io"
	"strconv"

	loggingclient "code.cloudfoundry.org/diego-logging-client"

	"github.com/cloudfoundry/sonde-go/events"
)

const (
	MAX_MESSAGE_SIZE = 61440

	DefaultLogSource = "LOG"
)

//go:generate counterfeiter -o fake_log_streamer/fake_log_streamer.go . LogStreamer
type LogStreamer interface {
	Stdout() io.Writer
	Stderr() io.Writer

	Flush()

	WithSource(sourceName string) LogStreamer
	SourceName() string
}

type logStreamer struct {
	stdout *streamDestination
	stderr *streamDestination
}

func New(guid string, sourceName string, index int, metronClient loggingclient.IngressClient) LogStreamer {
	if guid == "" {
		return noopStreamer{}
	}

	if sourceName == "" {
		sourceName = DefaultLogSource
	}

	sourceIndex := strconv.Itoa(index)

	return &logStreamer{
		stdout: newStreamDestination(
			guid,
			sourceName,
			sourceIndex,
			events.LogMessage_OUT,
			metronClient,
		),

		stderr: newStreamDestination(
			guid,
			sourceName,
			sourceIndex,
			events.LogMessage_ERR,
			metronClient,
		),
	}
}

func (e *logStreamer) Stdout() io.Writer {
	return e.stdout
}

func (e *logStreamer) Stderr() io.Writer {
	return e.stderr
}

func (e *logStreamer) Flush() {
	e.stdout.lockAndFlush()
	e.stderr.lockAndFlush()
}

func (e *logStreamer) WithSource(sourceName string) LogStreamer {
	if sourceName == "" {
		return e
	}

	return &logStreamer{
		stdout: e.stdout.withSource(sourceName),
		stderr: e.stderr.withSource(sourceName),
	}
}

func (e *logStreamer) SourceName() string {
	return e.stdout.sourceName
}
