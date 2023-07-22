package cereal

import (
	"errors"
	"io"

	"github.com/distributed/sers"
	goburrow "github.com/goburrow/serial"

	tarm "github.com/tarm/serial"
	bugst "go.bug.st/serial"
)

// Opener is an interface for working with serial port libraries to be able
// to easily interchange them.
//
// It is implemented by the various serial port libraries in this package for convenience.
type Opener interface {
	// OpenPort opens a serial port with the given name and mode.
	// portname is the name of the port to open, e.g. "/dev/ttyUSB0" or "COM1".
	OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error)
}

// Bugst implements the Opener interface for the go.bug.st/serial package.
type Bugst struct{}

func (Bugst) String() string      { return "bugst" }
func (Bugst) PackagePath() string { return "go.bug.st/serial" }

func (Bugst) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	return bugst.Open(portname, &bugst.Mode{
		BaudRate: mode.BaudRate,
		DataBits: mode.DataBits,
		Parity:   bugst.Parity(mode.Parity),
		StopBits: bugst.StopBits(mode.StopBits),
	})
}

// Tarm implements the Opener interface for the github.com/tarm/serial package.
type Tarm struct{}

func (Tarm) String() string      { return "tarm" }
func (Tarm) PackagePath() string { return "github.com/tarm/serial" }

func (Tarm) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	var parity tarm.Parity = tarm.Parity(mode.Parity.Char())
	return tarm.OpenPort(&tarm.Config{
		Name:   portname,
		Baud:   mode.BaudRate,
		Size:   byte(mode.DataBits),
		Parity: parity,
		StopBits: func() tarm.StopBits {
			switch mode.StopBits {
			case StopBits1:
				return tarm.Stop1
			case StopBits1Half:
				return tarm.Stop1Half
			case StopBits2:
				return tarm.Stop2
			default:
				return 0
			}
		}(),
	})
}

// Goburrow implements the Opener interface for the github.com/goburrow/serial package.
type Goburrow struct{}

func (Goburrow) String() string      { return "goburrow" }
func (Goburrow) PackagePath() string { return "github.com/goburrow/serial" }

func (Goburrow) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	if mode.StopBits == StopBits1Half {
		return nil, errors.New("unsupported stop bits")
	}
	return goburrow.Open(&goburrow.Config{
		Address:  portname,
		BaudRate: mode.BaudRate,
		DataBits: mode.DataBits,
		StopBits: mode.StopBits.Halves() / 2,
		Parity:   string(mode.Parity.Char()),
	})

}

// Sers implements the Opener interface for the github.com/distributed/sers package.
type Sers struct{}

func (Sers) String() string      { return "sers" }
func (Sers) PackagePath() string { return "github.com/distributed/sers" }

func (Sers) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	sp, err := sers.Open(portname)
	if err != nil {
		return nil, err
	}
	var parity, stopbits, databits int
	if databits == 0 {
		databits = 8
	}
	switch mode.Parity {
	case ParityNone:
		parity = sers.N
	case ParityOdd:
		parity = sers.O
	case ParityEven:
		parity = sers.E
	case ParityMark, ParitySpace:
		return nil, errors.New("unsupported parity")
	default:
		return nil, errors.New("invalid parity")
	}
	switch mode.StopBits {
	case StopBits1:
		stopbits = 1
	case StopBits2:
		stopbits = 2
	case StopBits1Half:
		return nil, errors.New("unsupported stop bits")
	default:
		return nil, errors.New("invalid stop bits")
	}
	err = sp.SetMode(mode.BaudRate, databits, parity, stopbits, sers.NO_HANDSHAKE)
	if err != nil {
		sp.Close() // ensure we close the port on error.
		return nil, err
	}
	return sp, nil
}
