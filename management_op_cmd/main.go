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
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"syscall"

	auth "github.com/arsonistgopher/junos-jet-demo-apps/proto/auth"
	mng "github.com/arsonistgopher/junos-jet-demo-apps/proto/management"
	"golang.org/x/crypto/ssh/terminal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// This is a cleanliness thing. Let's keep all the config data together.
type config struct {
	command    *string                  // Comamnd to send over RPC
	format     *string                  // Data format required (XML / JSON)
	host       *string                  // Hostname or IP address of Junos host
	port       *string                  // Port that the gRPC server is listening on
	user       *string                  // Username of Junos host
	clientid   *string                  // ClientID of session
	timeout    *int                     // Timeout of session in seconds
	passwd     *string                  // Password for user
	certdir    *string                  // Directory where certs are stored
	pbfmt      *mng.OperationFormatType // Format type to return in format check
	hoststring string                   // Full semi-colon tokensied string
}

func main() {

	log.Println("------------------------------")
	log.Println("Junos JET OpCommand Test Tool")
	log.Println("------------------------------")
	log.Print("Run the app with -h for options\n\n")

	// Create config instance
	var cfg config
	cfg = config{}
	cfg.pbfmt = new(mng.OperationFormatType)

	// Gather the config data including password from the terminal
	cfg.command = flag.String("command", "show version", "Operational command")
	cfg.format = flag.String("format", "xml", "XML or JSON")
	cfg.host = flag.String("host", "127.0.0.1", "Hostname or IP Address")
	cfg.port = flag.String("port", "32767", "Port that the grpc server is listening on.")
	cfg.user = flag.String("user", "jet", "Username for authentication")
	cfg.clientid = flag.String("cid", "42", "Client ID for session")
	cfg.timeout = flag.Int("timeout", 10, "Timeout in seconds for JET")
	cfg.passwd = flag.String("passwd", "", "Password for Junos host. Note, not mandatory")
	cfg.certdir = flag.String("certdir", "", "Directory with client.crt, client.key, CA.crt")
	flag.Parse()

	// Generate host string in pattern "host:port"
	cfg.hoststring = *cfg.host + ":" + *cfg.port

	// Grab password if not set
	if *cfg.passwd == "" {
		log.Print("Enter Password: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Err: %v\n", err)
		}
		*cfg.passwd = string(bytePassword)
	}

	// Next, check for XML vs JSON vs CLI
	switch strings.ToUpper(*cfg.format) {
	case "XML":
		*cfg.pbfmt = mng.OperationFormatType_OPERATION_FORMAT_XML
	case "JSON":
		*cfg.pbfmt = mng.OperationFormatType_OPERATION_FORMAT_JSON
	case "CLI":
		*cfg.pbfmt = mng.OperationFormatType_OPERATION_FORMAT_CLI
	default:
		log.Println("Unrecognised format type. Defaulting to XML")
		*cfg.pbfmt = mng.OperationFormatType_OPERATION_FORMAT_XML
	}

	// gRPC options
	var opts []grpc.DialOption

	// Are we going to run with TLS?
	runningWithTLS := false
	if *cfg.certdir != "" {
		runningWithTLS = true
	}

	// If we're running with TLS
	if runningWithTLS {

		// Grab x509 cert/key for client
		cert, err := tls.LoadX509KeyPair(fmt.Sprintf("%s/client.crt", *cfg.certdir), fmt.Sprintf("%s/client.key", *cfg.certdir))

		if err != nil {
			log.Fatalf("Could not load certFile: %v", err)
		}
		// Create certPool for CA
		certPool := x509.NewCertPool()

		// Get CA
		ca, err := ioutil.ReadFile(fmt.Sprintf("%s/CA.crt", *cfg.certdir))
		if err != nil {
			log.Fatalf("could not read ca certificate: %s", err)
		}

		// Append CA cert to pool
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			log.Fatal("Failed to append client certs")
		}

		// build creds
		creds := credentials.NewTLS(&tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{cert},
			ServerName:   *cfg.host,
		})

		if err != nil {
			log.Fatalf("Could not load clientCert: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else { // Else we're not running with TLS
		opts = append(opts, grpc.WithInsecure())
	}

	// Set up a connection to the server.
	// This script for ease tests without TLS. For production systems, do not do this.
	conn, err := grpc.Dial(cfg.hoststring, opts...)

	if err != nil {
		log.Fatalf("Did not connect: %v\n", err)
	}

	defer conn.Close()

	// Get new login client using our grpc conn
	c := auth.NewLoginClient(conn)

	// Perform login check against the JET auth API
	r, err := c.LoginCheck(context.Background(), &auth.LoginRequest{
		UserName: *cfg.user,
		Password: *cfg.passwd,
		ClientId: *cfg.clientid,
	})

	if err != nil {
		log.Fatalf("Could not connect. Check IP address or domain name: %v\n", err)
	} else {
		if r.GetResult() {
			log.Printf("Connect to %s successful\n", *cfg.host)
		}
	}

	// Now we have to create the management client
	mgmtc := mng.NewManagementRpcApiClient(conn)

	// Next, create the command to execute over RPC
	mngCmd := &mng.ExecuteOpCommandRequest_CliCommand{
		CliCommand: *cfg.command,
	}

	// Issue the request
	req := &mng.ExecuteOpCommandRequest{
		RequestId: uint64(42),
		Command:   mngCmd,
		OutFormat: *cfg.pbfmt,
	}

	// Execute the RPC and return the opclient
	opclient, err := mgmtc.ExecuteOpCommand(context.Background(), req)

	if err != nil {
		log.Fatal("Issue getting client for ExecuteOpCommand()")
	}

	// Block and recv()
	resp, err := opclient.Recv()

	if err != nil {
		log.Fatalf("Issue receiving data from Junos via gRPC: %s", err)
	}

	// Print the data vanity head and data itself
	fmt.Print("\n---Data---\n\n")
	fmt.Print(resp.GetData())
}
