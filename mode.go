package cereal

import (
	"errors"
	"time"

	bugst "go.bug.st/serial"
)

// Mode is the configuration for the serial port.
type Mode struct {
	BaudRate int
	// DataBits 5, 6, 7, 8. If Zero then 8 is used.
	DataBits int
	// ReadTimeout is the maximum time to wait for a read to complete.
	// May not be implemented on all platforms or Opener implementations.
	ReadTimeout time.Duration
	Parity      Parity
	StopBits    StopBits
}

var (
	errReadTimeoutUnsupportedBugst = errors.New("read timeout not supported for Opener implementation. Use a different Opener")
)

// StopBits is the number of stop bits to use- is a enum so use package defined
// StopBits1, StopBits1Half, StopBits2.
type StopBits byte

const (
	StopBits1     = StopBits(bugst.OneStopBit)
	StopBits1Half = StopBits(bugst.OnePointFiveStopBits)
	StopBits2     = StopBits(bugst.TwoStopBits)
)

// String returns a human readable representation of the stop bits.
func (s StopBits) String() (str string) {
	switch s {
	case StopBits1:
		str = "1"
	case StopBits1Half:
		str = "1.5"
	case StopBits2:
		str = "2"
	default:
		str = "Unknown"
	}
	return str
}

// Halves returns the number of half bits for the stop bits. If invalid returns 0.
func (s StopBits) Halves() (halves int) {
	switch s {
	case StopBits1:
		halves = 2
	case StopBits1Half:
		halves = 3
	case StopBits2:
		halves = 4
	}
	return halves
}

// Parity is the type of parity to use- is a enum so use package defined
// ParityNone, ParityOdd, ParityEven, ParityMark, ParitySpace.
type Parity byte

const (
	ParityNone  = Parity(bugst.NoParity)
	ParityOdd   = Parity(bugst.OddParity)
	ParityEven  = Parity(bugst.EvenParity)
	ParityMark  = Parity(bugst.MarkParity)
	ParitySpace = Parity(bugst.SpaceParity)
)

// String returns a human readable representation of the parity.
func (p Parity) String() (s string) {
	switch p {
	case ParityNone:
		s = "None"
	case ParityOdd:
		s = "Odd"
	case ParityEven:
		s = "Even"
	case ParityMark:
		s = "Mark"
	case ParitySpace:
		s = "Space"
	default:
		s = "Unknown"
	}
	return s
}

func (p Parity) Char() (char byte) {
	if p > ParitySpace {
		return '?'
	}
	return []byte{'N', 'O', 'E', 'M', 'S'}[p]
}
