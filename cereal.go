package cereal

import (
	"errors"
	"io"
	"strconv"

	"github.com/distributed/sers"
	goburrow "github.com/goburrow/serial"

	tarm "github.com/tarm/serial"
	bugst "go.bug.st/serial"
	"go.bug.st/serial/enumerator"
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

// PortDetails contains OS provided information on a USB or Serial port.
type PortDetails struct {
	Name     string
	VID, PID uint16
	IsUSB    bool
}

// ForEachPort calls the given function for each serial port found.
//
// ForEachPort returns early with fn's error if fn returns an error or
// if halt is true.
func ForEachPort(fn func(details PortDetails) (halt bool, err error)) error {
	detailedList, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return err
	}

	// Add missing non-detailed to the list of detailed ports. On windows COM ports may be missing.
	simpleList, err := bugst.GetPortsList()
	if err == nil {
		for _, portname := range simpleList {
			contained := false
			for _, detailedPort := range detailedList {
				if detailedPort.Name == portname {
					contained = true
					break
				}
			}
			if !contained {
				detailedList = append(detailedList, &enumerator.PortDetails{Name: portname})
			}
		}
	}
	for _, port := range detailedList {
		vid, _ := strconv.ParseUint(port.VID, 16, 16)
		pid, _ := strconv.ParseUint(port.PID, 16, 16)
		halt, err := fn(PortDetails{
			Name:  port.Name,
			VID:   uint16(vid),
			PID:   uint16(pid),
			IsUSB: port.IsUSB,
		})
		if err != nil || halt {
			return err
		}
	}
	return nil
}

// Bugst implements the Opener interface for the go.bug.st/serial package.
type Bugst struct{}

func (Bugst) String() string      { return "bugst" }
func (Bugst) PackagePath() string { return "go.bug.st/serial" }

func (Bugst) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	if mode.ReadTimeout != 0 {
		return nil, errReadTimeoutUnsupportedBugst
	}
	var parity bugst.Parity
	switch mode.Parity {
	case ParityNone:
		parity = bugst.NoParity
	case ParityOdd:
		parity = bugst.OddParity
	case ParityEven:
		parity = bugst.EvenParity
	case ParityMark:
		parity = bugst.MarkParity
	case ParitySpace:
		parity = bugst.SpaceParity
	default:
		return nil, errInvalidParity
	}

	var stopbits bugst.StopBits
	switch mode.StopBits {
	case StopBits1:
		stopbits = bugst.OneStopBit
	case StopBits1Half:
		stopbits = bugst.OnePointFiveStopBits
	case StopBits2:
		stopbits = bugst.TwoStopBits
	default:
		return nil, errInvalidStopbits
	}
	return bugst.Open(portname, &bugst.Mode{
		BaudRate: mode.BaudRate,
		DataBits: mode.DataBits,
		Parity:   parity,
		StopBits: stopbits,
	})
}

// Tarm implements the Opener interface for the github.com/tarm/serial package.
type Tarm struct{}

func (Tarm) String() string      { return "tarm" }
func (Tarm) PackagePath() string { return "github.com/tarm/serial" }

func (Tarm) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	var parity tarm.Parity = tarm.Parity(mode.Parity.Char())
	return tarm.OpenPort(&tarm.Config{
		Name:        portname,
		Baud:        mode.BaudRate,
		Size:        byte(mode.DataBits),
		Parity:      parity,
		ReadTimeout: mode.ReadTimeout,
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
		return nil, errUnsupportedStopbits
	}
	return goburrow.Open(&goburrow.Config{
		Address:  portname,
		BaudRate: mode.BaudRate,
		DataBits: mode.DataBits,
		StopBits: mode.StopBits.Halves() / 2,
		Parity:   string(mode.Parity.Char()),
		Timeout:  mode.ReadTimeout,
	})

}

// Sers implements the Opener interface for the github.com/distributed/sers package.
type Sers struct{}

func (Sers) String() string      { return "sers" }
func (Sers) PackagePath() string { return "github.com/distributed/sers" }

func (Sers) OpenPort(portname string, mode Mode) (io.ReadWriteCloser, error) {
	sp, err := openSers(portname)
	if err != nil {
		return nil, err
	}
	if mode.ReadTimeout != 0 {
		err = sp.SetReadParams(0, mode.ReadTimeout.Seconds())
		if err != nil {
			return nil, err
		}
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
		return nil, errUnsupportedParity
	default:
		return nil, errInvalidParity
	}
	switch mode.StopBits {
	case StopBits1:
		stopbits = 1
	case StopBits2:
		stopbits = 2
	case StopBits1Half:
		return nil, errUnsupportedStopbits
	default:
		return nil, errInvalidStopbits
	}
	err = sp.SetMode(mode.BaudRate, databits, parity, stopbits, sers.NO_HANDSHAKE)
	if err != nil {
		sp.Close() // ensure we close the port on error.
		return nil, err
	}
	return sp, nil
}

// ResetInputBuffer discards data received but not read by the port. It expects a port type
// or an interface that implements `Reset()`/`Reset() error`/`ResetInputBuffer() error`. An error is returned
// if the functionality is not implemented by the port.
func ResetInputBuffer(port io.Reader) error {
	// Test for common ports
	switch r := port.(type) {
	case sers.SerialPort, *tarm.Port, goburrow.Port:
		return errors.New("cereal: sers/tarm/goburrow does not support ResetInputBuffer")
	case bugst.Port:
		return r.ResetInputBuffer()
	case *NonBlocking:
		r.Reset()
		return nil
	}
	type resetter interface {
		Reset()
	}
	type resetterErr interface {
		Reset() error
	}
	type resetInputBuffer interface {
		ResetInputBuffer() error
	}
	if r, ok := port.(resetter); ok {
		r.Reset()
		return nil
	} else if r, ok := port.(resetterErr); ok {
		return r.Reset()
	} else if r, ok := port.(resetInputBuffer); ok {
		return r.ResetInputBuffer()
	}
	return errors.New("cereal: ResetInputBuffer not implemented by argument")
}
