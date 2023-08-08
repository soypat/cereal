//go:build !(arm && !cgo)

package cereal

import "github.com/distributed/sers"

func openSers(portname string) (sers.SerialPort, error) {
	return sers.Open(portname)
}
