package cereal_test

import (
	"flag"
	"log"
	"time"

	"github.com/soypat/cereal"
)

func ExampleOpener() {
	availableLibs := map[string]cereal.Opener{
		cereal.Bugst{}.String():    cereal.Bugst{},
		cereal.Tarm{}.String():     cereal.Tarm{},
		cereal.Goburrow{}.String(): cereal.Goburrow{},
		cereal.Sers{}.String():     cereal.Sers{},
	}
	flagSerial := flag.String("seriallib", "bugst", "Serial library to use: bugst, tarm, goburrow, sers")
	flag.Parse()
	serial, ok := availableLibs[*flagSerial]
	if !ok {
		flag.PrintDefaults()
		log.Fatalf("Invalid serial library: %s\n", *flagSerial)
	}

	port, err := serial.OpenPort("/dev/ttyUSB0", cereal.Mode{
		BaudRate: 9600,
		DataBits: 8,
		Parity:   cereal.ParityNone,
		StopBits: cereal.StopBits1,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer port.Close()

	// Do something with port
	readBuffer := make([]byte, 128)
	for {
		_, err := port.Write([]byte("Hello\n"))
		if err != nil {
			log.Fatal(err)
		}
		n, err := port.Read(readBuffer)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Read %d bytes: %q\n", n, string(readBuffer[:n]))
		time.Sleep(time.Second)
	}
}

func ExampleForEachPort() {
	err := cereal.ForEachPort(func(port cereal.PortDetails) (bool, error) {
		log.Printf("%v\n", port) // Log all port details.
		return false, nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
