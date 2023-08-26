package cereal_test

import (
	"bytes"
	"flag"
	"io"
	"log"
	"math/rand"
	"testing"
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

func TestNonBlockingRead(t *testing.T) {
	t.Parallel()
	var data [1024]byte
	for i := range data {
		data[i] = byte(rand.Intn(256))
	}
	buf := bytes.NewBuffer(data[:])
	bbuf := nop{
		ReadWriter: buf,
		Closer:     io.NopCloser(buf),
	}

	nb := cereal.NewNonBlocking(bbuf, cereal.NonBlockingConfig{})

	smallbuf := make([]byte, 31)
	n := 0
	for n < len(data) {
		nn, err := nb.Read(smallbuf[:])
		got := smallbuf[:nn]
		expect := data[n : n+nn]
		n += nn
		if !bytes.Equal(got, expect) {
			t.Fatalf("mismatch in data read:\n%q\n%q", got, expect)
		}
		if err != nil && n != len(data) {
			t.Error(err)
			break
		}
	}
}

func TestNonBlockingBlocked(t *testing.T) {
	t.Parallel()
	const (
		block   = 100 * time.Millisecond
		timeout = time.Millisecond
		data    = "hello partner!"
	)
	rwc := &readwritecloser{
		read: func(b []byte) (int, error) {
			time.Sleep(block)
			return copy(b, data), nil
		},
	}

	nb := cereal.NewNonBlocking(rwc, cereal.NonBlockingConfig{
		Timeout: timeout,
	})
	// This call should fail with deadline exceeded.
	buf := make([]byte, len(data))
	n, err := nb.Read(buf)
	if n != 0 || err == nil {
		t.Fatal("unexpected NonBlocking behaviour", n, err)
	}
	time.Sleep(block - timeout)
	n, err = nb.Read(buf)
	if n != len(buf) || err != nil {
		t.Fatal("expected to read blocked data", n, err)
	}

	if string(buf) != data {
		t.Errorf("expected %q; got %q", data, buf)
	}
}

func TestNonBlockingReset(t *testing.T) {
	t.Parallel()
	const (
		block   = 100 * time.Millisecond
		timeout = time.Millisecond
		data    = "hello partner!"
	)
	rwc := &readwritecloser{
		read: func(b []byte) (int, error) {
			time.Sleep(block)
			return copy(b, data), nil
		},
	}

	nb := cereal.NewNonBlocking(rwc, cereal.NonBlockingConfig{
		Timeout: timeout,
	})
	// This call should fail with deadline exceeded.
	buf := make([]byte, len(data))
	n, err := nb.Read(buf)
	if n != 0 || err == nil {
		t.Fatal("unexpected NonBlocking behaviour", n, err)
	}
	time.Sleep(block - timeout)
	nb.Reset()
	n, _ = nb.Read(buf)
	if n != 0 {
		t.Fatal("expected data to have been completely reset, got", string(buf[:n]))
	}
}

type nop struct {
	io.ReadWriter
	io.Closer
}

type readwritecloser struct {
	read, write func([]byte) (int, error)
	close       func() error
}

func (rwc *readwritecloser) Read(b []byte) (int, error) {
	if rwc.read == nil {
		panic("nill read callback")
	}
	return rwc.read(b)
}
func (rwc *readwritecloser) Write(b []byte) (int, error) {
	if rwc.write == nil {
		return len(b), nil
	}
	return rwc.write(b)
}

func (rwc *readwritecloser) Close() error {
	if rwc.close == nil {
		return nil
	}
	return rwc.close()
}
