//go:build !cgo

package cereal

import (
	"errors"

	"github.com/distributed/sers"
)

var serserr = errors.New("github.com/distributed/sers.OpenPort requires CGO")

func openSers(portname string) (sers.SerialPort, error) {
	return nil, serserr
}
