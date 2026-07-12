package ultrasound

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/syugeeeeeeeeeei/BRIDGE/src/bridge/bearing"
)

type EventSink interface {
	WriteEvent(context.Context, bearing.Event) error
	Close(context.Context) error
}

type DiscardSink struct{}

func (DiscardSink) WriteEvent(context.Context, bearing.Event) error { return nil }
func (DiscardSink) Close(context.Context) error                     { return nil }

type WriterSink struct {
	mu     sync.Mutex
	enc    *json.Encoder
	closed bool
}

func NewWriterSink(w io.Writer) *WriterSink { return &WriterSink{enc: json.NewEncoder(w)} }
func (s *WriterSink) WriteEvent(ctx context.Context, e bearing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("event sink is closed")
	}
	return s.enc.Encode(cloneEvent(e))
}
func (s *WriterSink) Close(context.Context) error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}

type FileSink struct {
	mu     sync.Mutex
	file   *os.File
	buf    *bufio.Writer
	enc    *json.Encoder
	closed bool
	path   string
}

func NewFileSink(path string, overwrite bool) (*FileSink, error) {
	if path == "" {
		return nil, errors.New("trace output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	flags := os.O_WRONLY | os.O_CREATE
	if overwrite {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, err
	}
	b := bufio.NewWriterSize(f, 256*1024)
	return &FileSink{file: f, buf: b, enc: json.NewEncoder(b), path: path}, nil
}
func (s *FileSink) Path() string { return s.path }
func (s *FileSink) WriteEvent(ctx context.Context, e bearing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("event sink is closed")
	}
	return s.enc.Encode(cloneEvent(e))
}
func (s *FileSink) Close(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if err := s.buf.Flush(); err != nil {
		_ = s.file.Close()
		return err
	}
	return s.file.Close()
}

type MemorySink struct {
	mu     sync.Mutex
	events []bearing.Event
	closed bool
}

func (s *MemorySink) WriteEvent(ctx context.Context, e bearing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return errors.New("event sink is closed")
	}
	s.events = append(s.events, cloneEvent(e))
	return nil
}
func (s *MemorySink) Close(context.Context) error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}
func (s *MemorySink) Events() []bearing.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]bearing.Event, len(s.events))
	copy(out, s.events)
	return out
}

type CallbackSink struct{ Callback func(bearing.Event) error }

func (s CallbackSink) WriteEvent(ctx context.Context, e bearing.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Callback == nil {
		return nil
	}
	return s.Callback(cloneEvent(e))
}
func (s CallbackSink) Close(context.Context) error { return nil }

type MultiSink struct{ Sinks []EventSink }

func (s MultiSink) WriteEvent(ctx context.Context, e bearing.Event) error {
	for _, x := range s.Sinks {
		if x != nil {
			if err := x.WriteEvent(ctx, e); err != nil {
				return err
			}
		}
	}
	return nil
}
func (s MultiSink) Close(ctx context.Context) error {
	var first error
	for _, x := range s.Sinks {
		if x != nil {
			if err := x.Close(ctx); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}
