# cereal
Serial port abstraction creation for bugst, sers, goburrow and tarm serial libraries.

This can make it easier to diagnose if a bug is an issue with a certain library or not.

## Example

Below is a program that writes and reads from a serial port.

The library used to access the port is selected by the program user via a flag.

```sh
program -seriallib=tarm
```

```go
package main

import (
	"flag"
	"log"
	"time"

	"github.com/soypat/cereal"
)

func main() {
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
```