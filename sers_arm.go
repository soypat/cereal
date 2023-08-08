//go:build arm && !cgo

package cereal

import (
	"errors"

	"github.com/distributed/sers"
)

var serserr = errors.New("github.com/distributed/sers.OpenPort not supported for ARM without CGo")

func openSers(portname string) (sers.SerialPort, error) {
	return nil, serserr
}
