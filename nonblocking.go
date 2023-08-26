package cereal

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"
)

var _ io.ReadWriteCloser = &NonBlocking{}

var (
	errDeadlineExceeded = errors.New("blocking deadline exceeded")
)

// NonBlocking implements io.Reader non-blocking behaviour. This is particular functionality is suited
// when developing message-based protocols over serial communication.
//
// A text-book example of a NonBlocking use case would be if one has multiple USB/Serial devices connected
// to a computer and one must write and read each to identify each device. If one device does not respond
// it will block on the Read call. If each device is wrapped with a NonBlocking and a timeout is set
// then the user can expect all Read calls to terminate withing the deadline/timeout given.
type NonBlocking struct {
	io             io.ReadWriteCloser
	defaultTimeout time.Duration
	maxBuffered    int
	mu             sync.Mutex
	buf            bytes.Buffer
	errfield       error
}

// NonBlockingConfig is used to configure the creation of a NonBlocking instance.
type NonBlockingConfig struct {
	// Timeout will define the timeout to wait on a Read call before returning deadline exceeded error.
	// If Timeout is zero then Read calls will return immediately and only have an error if the Reader
	// was closed or EOFed.
	Timeout time.Duration
	// MaxBuffered specifies the maximum amount of bytes to have buffered in our reader.
	// After MaxBuffered is reached a NonBlocking will sleep until the caller has read bytes
	// and made space for more reads. If set to zero no limit will be placed on buffer size.
	MaxBuffered int
}

// NewNonBlocking creates a [NonBlocking] instance with the given configuration parameters.
// To manage the non-blocking behaviour NewNonBlocking creates a goroutine which lives until
// the reader returns io.EOF or Close is called on NonBlocking.
func NewNonBlocking(rwc io.ReadWriteCloser, cfg NonBlockingConfig) *NonBlocking {
	if rwc == nil {
		panic("nil ReadWriteCloser passed into NewNonBlocking")
	}
	if cfg.Timeout < 0 || cfg.MaxBuffered < 0 {
		panic("invalid argument to NewNonBlocking")
	}
	nb := &NonBlocking{
		io:             rwc,
		defaultTimeout: cfg.Timeout,
		maxBuffered:    cfg.MaxBuffered,
	}
	go func() {
		const busySleep = 256 * time.Millisecond
		var buf [1024]byte
		for nb.err() == nil {
			if nb.maxBuffered != 0 && nb.Buffered() > nb.maxBuffered {
				time.Sleep(busySleep) // This busy sleep is to not blow up our buffer.
				continue
			}
			n, err := nb.io.Read(buf[:])
			if err != nil && errors.Is(err, io.EOF) {
				nb.mu.Lock()
				nb.buf.Write(buf[:n])
				nb.mu.Unlock()
				nb.setErr(err) // Our Reader is done. Nothing more to do here.
				return
			}
			if n == 0 {
				time.Sleep(busySleep) // An empty read is a good indicator that nothing much is happening on bus, so sleep.
				continue
			}
			nb.mu.Lock()
			nb.buf.Write(buf[:n])
			nb.mu.Unlock()
		}
	}()
	return nb
}

// Write implements the [io.Writer] interface. Sends writes directly to the underlying Writer.
func (nb *NonBlocking) Write(b []byte) (int, error) {
	return nb.io.Write(b)
}

// Read implements the [io.Reader] interface. Will call NonBlocking.ReadDeadline with the set timeout.
func (nb *NonBlocking) Read(b []byte) (int, error) {
	if nb.defaultTimeout == 0 {
		// Fast track for no-timeouts configuration.
		nb.mu.Lock()
		defer nb.mu.Unlock()
		n, _ := nb.buf.Read(b)
		return n, nb.errfield
	}
	deadline := time.Now().Add(nb.defaultTimeout)
	return nb.ReadDeadline(b, deadline)
}

// ReadDeadline reads from the underlying buffer up until the deadline.
func (nb *NonBlocking) ReadDeadline(b []byte, deadline time.Time) (n int, err error) {
	for err == nil && n < len(b) {
		var nn int
		nn, err = nb.readNext(b[n:], deadline)
		n += nn
	}
	if nb.err() != nil && n == 0 && err == nil {
		// Early setting of the error if the reader has failed and no more bytes are being read.
		// This means that the reader is likely done.
		err = nb.err()
	}
	return n, err
}

func (nb *NonBlocking) readNext(b []byte, deadline time.Time) (int, error) {
	n := nb.Buffered()
	for n <= 0 {
		until := time.Until(deadline)
		if until < 0 {
			return 0, errDeadlineExceeded
		} else if err := nb.err(); err != nil {
			return 0, err // Our reader failed, no recovery so just exit.
		}
		time.Sleep(minD(100*time.Millisecond, until))
		n = nb.Buffered()
	}
	nb.mu.Lock()
	defer nb.mu.Unlock()
	if nb.buf.Len() == 0 {
		// There was a race to read buf and we lost.
		// This can happen if there are multiple callers to ReadDeadline.
		return 0, nil
	}
	// We ignore io.EOF returned by buffer since unless goroutine is done it is not really EOF.
	n, _ = nb.buf.Read(b)
	return n, nil
}

// Buffered returns the amount of bytes in the underlying buffer.
func (nb *NonBlocking) Buffered() int {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	return nb.buf.Len()
}

// Close terminates to reader and writer. Sets [io.EOF] as the returned error for future Read calls.
func (nb *NonBlocking) Close() error {
	nb.setErr(io.EOF)
	return nb.io.Close()
}

// Reset resets the underlying buffer to be empty, discarding all data read.
// Reset is useful for message-based protocols where a slow response that timed out
// can be interpreted as a response to the next call to Read.
func (nb *NonBlocking) Reset() {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	nb.buf.Reset()
}

// err returns error set by setErr. If err is set read goroutine is done or in process of ending.
func (nb *NonBlocking) err() error {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	return nb.errfield
}

func (nb *NonBlocking) setErr(err error) {
	nb.mu.Lock()
	defer nb.mu.Unlock()
	nb.errfield = err
}

func minD(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
