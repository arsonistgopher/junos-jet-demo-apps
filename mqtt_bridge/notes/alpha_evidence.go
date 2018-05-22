// TODO: License

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sevlyar/go-daemon"
)

// CLID is a temporary CLID
const CLID = "junos-jet-bridge"

// VERSION is a version string
const VERSION = "0.1"

// Prototype for func call: createListener(HOST, CLID, TOPIC, PID, CHAN, WG)
func createListener(HOST string, CLID string, TOPIC string, PID int, DONE chan bool, WG *sync.WaitGroup) {

	go func(HOST string, CLID string, TOPIC string, HEARTBEAT chan bool, WG *sync.WaitGroup) {

		opts := mqtt.NewClientOptions().AddBroker(HOST).SetClientID(CLID)

		client := mqtt.NewClient(opts)
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.Fatal(token.Error())
		}

		if token := client.Subscribe(TOPIC, 0, func(client mqtt.Client, msg mqtt.Message) {
			smsg := string(msg.Payload())
			log.Print("Received message: ", smsg)

			// 'logger' is the application that gives us the ability to send logs in to Junos.
			// I did look at creating the serialisation for posting directly to eventd, but figured this:
			// a) The underlying logging system could change and 'logger' is likely to be up to date
			// b) As a result of a) I have to make fewer changes to this!
			binary, lookErr := exec.LookPath("logger")
			if lookErr != nil {
				panic(lookErr)
			}

			daemon := fmt.Sprintf("gojetmqttbridge[%v]", PID)

			// Args that we're passing to logger
			args := []string{"-d", daemon, "-e", "MSG_RECVD", smsg}

			// Obtain a cmd struct to execute the named program with the given arguments.
			cmd := exec.Command(binary, args...)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				log.Fatal(err)
			}

			/*
				raddr := net.UnixAddr{Name: "/var/run/eventd_events", Net: "unixgram"}
				conn, err := net.DialUnix("unixgram", nil, // can be nil
					&raddr)
				if err != nil {
					log.Panic("Dial error: ", err)
				}

				// Here we start building out the TLV!!!
				type hdr struct {
					emhLength  uint32 // Calculate this later
					emhVersion int16  // 4 for Junos
					eventType  int16  // 2 for syslog
					res1       int16  // *shoulder shrug* dunno what this is for...
				}

				hdtTest := &hdr{emhLength: uint32(0), emhVersion: int16(4), eventType: int16(2), res1: int16(0)}

				buf := new(bytes.Buffer)

				binary.Write(buf, binary.LittleEndian, hdtTest.emhLength)

				binary.Write(buf, binary.LittleEndian, hdtTest.emhVersion)

				binary.Write(buf, binary.BigEndian, hdtTest.eventType)

				binary.Write(buf, binary.BigEndian, hdtTest.res1)

				log.Print(buf.Bytes())

				// This test checks if the header makes it across!
				_, err = conn.Write(buf.Bytes())
				if err != nil {
					panic(err)
				}
				conn.Close()*/

		}); token.Wait() && token.Error() != nil {
			log.Fatal(token.Error())
			os.Exit(1)
		}

		for {
			select {
			case _, ok := <-DONE:
				if ok == false {
					// Handle Unsubscribe
					if token := client.Unsubscribe(TOPIC); token.Wait() && token.Error() != nil {
						log.Println(token.Error())
					}
					// Now signal we're done
					WG.Done()

					// Disconnect from MQTT
					client.Disconnect(250)
					// Return from Go Routine
					return
				}
			}
		}

		// If we're done then go through this,

	}(HOST, CLID, TOPIC, DONE, WG)
}

func main() {

	host := flag.String("host", "127.0.0.1", "Host IP address or hostname")
	port := flag.String("port", "1883", "Port of MQTT listener")
	topic := flag.String("topic", "junos/MQTTBridge", "Topic for subscribing to MQTT")
	flag.Parse()

	fullhost := *host + ":" + *port

	cntxt := &daemon.Context{
		PidFileName: "pid",
		PidFilePerm: 0644,
		LogFileName: "log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"junos-jet-mqtt-bridge"},
	}

	d, err := cntxt.Reborn()
	if err != nil {
		log.Print("Unable to run: ", err)
		// Do not loiter, exit with 1
		os.Exit(1)
	}
	if d != nil {
		return
	}

	/*lf, err := NewLogFile(cntxt.LogFileName, os.Stderr)
	if err != nil {
		log.Fatal("Unable to create log file: ", err)
	}
	log.SetOutput(lf)
	// rotate log every 24 hours
	rotateLogSignal := time.Tick(24 * time.Hour)
	go func() {
		for {
			select {
			case <-rotateLogSignal:

				if err := lf.Rotate(); err != nil {
					log.Fatal("Unable to rotate log: ", err)
				}
			}
		}
	}()
	// We get to here, we know we're the new child (from a foking point of view)
	defer cntxt.Release()
	*/

	var WG sync.WaitGroup
	WG.Add(1)
	PID := os.Getpid()

	// Create signal channel and register signals of interest
	sigs := make(chan os.Signal, 1)
	DONE := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create Go Routine listener (Go Routine happens in func)
	createListener(fullhost, CLID, *topic, PID, DONE, &WG)

	// If we get a signal, send false over HB channel
	go func() {
		for {

			select {
			case c := <-sigs:

				if c == syscall.SIGINT || c == syscall.SIGTERM {
					// If we move to here, we know we've got a signal
					// also, tell our other channel in the main Go routine that we're done
					DONE <- true
					close(DONE)
					return
				}
			}
		}
	}()

	log.Print("Starting Version: ", VERSION)

	for {

		select {
		case <-DONE:
			// Wait here for things to finish properly
			WG.Wait()
			os.Exit(0)
		}
	}
}
