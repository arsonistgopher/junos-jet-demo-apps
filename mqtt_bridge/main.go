/*
Copyright 2018 David Gee, Juniper Networks
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sevlyar/go-daemon"
)

// CLID is a temporary CLID
const CLID = "junos-jet-bridge"

// VERSION is a version string
const VERSION = "0.1a"

// Prototype for func call: createListener(HOST, CLID, TOPIC, PID, CHAN, WG)
func createListener(HOST string, CLID string, TOPIC string, PID int, DONE chan bool, WG *sync.WaitGroup) {
	log.Println("Connect to ", HOST)

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
			// b) As a result of a) I have to make fewer changes to this, so result?

			binary, lookErr := exec.LookPath("logger")
			if lookErr != nil {
				log.Print("Cannot find logger: ", lookErr)
				os.Exit(1)
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
				log.Fatal("[ERROR] executing command for 'logger': ", err)
			}

		}); token.Wait() && token.Error() != nil {
			log.Fatal("[ERROR] issue with token", token.Error())
		}

		for {
			select {
			case <-DONE:
				// Handle Unsubscribe
				if token := client.Unsubscribe(TOPIC); token.Wait() && token.Error() != nil {
					log.Println(token.Error())
				}

				// Disconnect from MQTT
				client.Disconnect(250)
				// Return from Go Routine

				// Now signal we're done
				WG.Done()
				return
			}
		}

	}(HOST, CLID, TOPIC, DONE, WG)
}

var (
	host  = flag.String("host", "127.0.0.1", "Host IP address or hostname")
	port  = flag.String("port", "1883", "Port of MQTT listener")
	topic = flag.String("topic", "junos/MQTTBridge", "Topic for subscribing to MQTT")
)

func main() {

	flag.Parse()

	cntxt := &daemon.Context{
		PidFileName: "pid",
		PidFilePerm: 0644,
		LogFileName: "log",
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"junos-jet-mqtt-bridge"},
	}

	fullhost := "tcp://" + *host + ":" + *port

	// Argument [1]
	cntxt.Args = append(cntxt.Args, fullhost)
	// Argument [2]
	cntxt.Args = append(cntxt.Args, *topic)

	d, err := cntxt.Reborn()
	if err != nil {
		log.Print("Unable to run: ", err)
		// Do not loiter, exit with 1
		os.Exit(1)
	}
	if d != nil {
		return
	}

	fmt.Println(fullhost)

	lf, err := NewLogFile(cntxt.LogFileName, os.Stderr)
	if err != nil {
		log.Fatal("Unable to create log file: ", err)
	}
	log.SetOutput(lf)
	// rotate log every 24 hours
	rotateLogSignal := time.Tick(24 * time.Hour)

	// Create a WaitGroup for various GRs
	var WG sync.WaitGroup

	// Log rotate channel for signalling it to close
	lrchan := make(chan bool, 1)

	WG.Add(1)
	go func(lrc chan bool) {
		for {
			select {
			case <-rotateLogSignal:
				if err := lf.Rotate(); err != nil {
					log.Fatal("Unable to rotate log: ", err)
				}
			case <-lrc:
				log.Println("Received kill signal for log rotation GR")
				WG.Done()
				return
			}
		}
	}(lrchan)

	// We get to here, we know we're the new child (from a forking point of view)
	defer cntxt.Release()

	PID := os.Getpid()

	// Create signal channel and register signals of interest
	sigs := make(chan os.Signal, 1)
	sigDeath := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create Go Routine listener (Go Routine happens in func). We might want many listeners...
	// I did think about passing in a channel for receiving launch errors, but the createListener func handles errors internally and will cause an exit. Bad coding?
	// I went for succinct over my version of correctness. Internal error handling could be done much better, but alas, it works and is readable.
	createListener(cntxt.Args[1], CLID, cntxt.Args[2], PID, sigDeath, &WG)

	// Create signal listener loop GR
	go func() {
		for {

			select {
			case c := <-sigs:

				if c == syscall.SIGINT || c == syscall.SIGTERM {
					// If we move to here, we know we've got a signal we need to do something about
					// If our GR list starts to grow, this needs to be in a slice and we should iterate through them
					lrchan <- true
					// Signal for Other GRs to exit

					WG.Wait()

					// Ok, everything else exited
					// Signal to main
					sigDeath <- true
					return
				}
			}
		}
	}()

	log.Print("Starting Version: ", VERSION)

	// All setup has been done, so wait here until we receive a "death" signal from our GR
	for {
		select {
		case <-sigDeath:
			log.Println("Main loop sigDeath<- signalled and clear to exit")
			// We are clear to die. Let's die with honour *bleurgh*
			close(sigDeath)
			os.Exit(0)
		}
	}
}
