package cereal

import (
	"errors"
	"time"
)

// Mode is the configuration for the serial port.
type Mode struct {
	BaudRate int
	// DataBits 5, 6, 7, 8. If Zero then 8 is used.
	DataBits int
	// ReadTimeout is the maximum time to wait for a read to complete.
	// May not be implemented on all platforms or Opener implementations.
	//
	// This value corresponds to VTIM in termios implementations.
	ReadTimeout time.Duration
	Parity      Parity
	StopBits    StopBits
}

var (
	errReadTimeoutUnsupportedBugst = errors.New("read timeout not supported for Opener implementation. Use a different Opener")
	errUnsupportedStopbits         = errors.New("stop bits unsupported")
	errInvalidStopbits             = errors.New("invalid stop bits")

	errUnsupportedParity = errors.New("unsupported parity")
	errInvalidParity     = errors.New("invalid parity")
)

// StopBits is the number of stop bits to use- is a enum so use package defined
// StopBits1, StopBits1Half, StopBits2.
type StopBits byte

const (
	StopBits1 StopBits = iota
	StopBits1Half
	StopBits2
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
		str = "<invalid stopbits>"
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
	return 0
}

// Parity is the type of parity to use- is a enum so use package defined
// ParityNone, ParityOdd, ParityEven, ParityMark, ParitySpace.
type Parity byte

const (
	ParityNone Parity = iota
	ParityOdd
	ParityEven
	ParityMark
	ParitySpace
)

var parityTable = [...]string{
	ParityNone:  "None",
	ParityOdd:   "Odd",
	ParityEven:  "Even",
	ParityMark:  "Mark",
	ParitySpace: "Space",
}

// String returns a human readable representation of the parity.
func (p Parity) String() (s string) {
	if int(p) >= len(parityTable) || parityTable[p] == "" {
		return "<invalid parity>"
	}
	return parityTable[p]
}

func (p Parity) Char() (char byte) {
	str := p.String()
	if str[0] == '<' {
		return '?'
	}
	return str[0]
}
