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
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	auth "github.com/arsonistgopher/junos-jet-demo-apps/proto/auth"
	routing "github.com/arsonistgopher/junos-jet-demo-apps/proto/bgp_route"
	jnxType "github.com/arsonistgopher/junos-jet-demo-apps/proto/jnx_addr"
	prpd "github.com/arsonistgopher/junos-jet-demo-apps/proto/prpd_common"
	"golang.org/x/crypto/ssh/terminal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// reqCookie is a const for requesting a unique cookie
	reqCookie = uint8(0) // Cookie
	add       = 0        // Verb for add
	del       = 1        // Verb for delete
)

// custom struct route type for loading our configuration based routes.
type route struct {
	Prefix   string   `toml:"prefix"`
	Length   uint32   `toml:"length"`
	NextHops []string `toml:"nexthops"`
}

// custom struct route type for loading our configuration based routes.
type basics struct {
	LocalPref  uint32 `toml:"localPref"`
	RoutePref  uint32 `toml:"routePref"`
	AsPathStr  string `toml:"asPathStr"`
	Originator string `toml:"originator"`
	Cluster    string `toml:"cluster"`
}

// TOML based config struct for loading from a configuration file.
type routes struct {
	Basics basics
	Routes []route `toml:"route"`
}

// This is a cleanliness thing. Let's keep all the config data together.
type config struct {
	routesfile *string // Location of file with routes
	format     *string // Data format required (XML / JSON)
	host       *string // Hostname or IP address of Junos host
	port       *string // Port that the gRPC server is listening on
	user       *string // Username of Junos host
	clientid   *string // ClientID of session
	timeout    *int    // Timeout of session in seconds
	passwd     *string // Password for user
	certdir    *string // Directory where certs are stored
	verb       *string // Verb, add or delete
	hoststring string  // Full semi-colon tokensied string
}

// getCookie() returns a unique cookie using channels.
// This runs as a go routine and responses are returned in a channel.
// This pattern means you do not have to use atomic types or use mutexes to protect counters etc.
func getCookie() (chan uint8, chan uint64) {
	req := make(chan uint8)
	res := make(chan uint64)
	// Initalise a cookie with a preset value
	var pathCookie uint64 = 12345678

	// Launch the go routine
	go func(req chan uint8, res chan uint64) {
		for {
			select {

			case r := <-req:
				if r == reqCookie {
					pathCookie++
					res <- pathCookie
				}
			}
		}
	}(req, res)

	// Return the means to talk to the go routine and protected cookie var!
	return req, res
}

// This function takes the hard work out of getting a RoutePrefix instance
func getInetPrefix(s string) *prpd.RoutePrefix {
	inetAddr := &jnxType.IpAddress{AddrFormat: &jnxType.IpAddress_AddrString{AddrString: s}}
	inetPrefixInet := &prpd.RoutePrefix_Inet{Inet: inetAddr}
	inetPrefix := &prpd.RoutePrefix{RoutePrefixAf: inetPrefixInet}
	return inetPrefix
}

func main() {
	log.Println("--------------------------------------")
	log.Println("Junos JET BGP-Static Route Test Client")
	log.Println("--------------------------------------")
	log.Print("Run the app with -h for options\n\n")

	// Create config instance
	var cfg config
	cfg = config{}

	// Gather the config data including password from the terminal
	cfg.routesfile = flag.String("routesfile", "routes.toml", "File containing routes")
	cfg.host = flag.String("host", "127.0.0.1", "Hostname or IP Address")
	cfg.port = flag.String("port", "32767", "Port that the grpc server is listening on.")
	cfg.user = flag.String("user", "jet", "Username for authentication")
	cfg.clientid = flag.String("cid", "42", "Client ID for session")
	cfg.timeout = flag.Int("timeout", 10, "Timeout in seconds for JET")
	cfg.passwd = flag.String("passwd", "", "Password for Junos host. Note, not mandatory")
	cfg.certdir = flag.String("certdir", "", "Directory with client.crt, client.key, CA.crt")
	cfg.verb = flag.String("verb", "add", "Verb is 'add' or 'del'")
	flag.Parse()

	// Grab password if not set. Do this first. Saves time if the user gets it wrong
	if *cfg.passwd == "" {
		log.Print("Enter Password: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Err: %v\n", err)
		}
		*cfg.passwd = string(bytePassword)
	}

	// Generate host string in pattern "host:port"
	cfg.hoststring = *cfg.host + ":" + *cfg.port

	// Oper is the operational verb: add/del routes
	oper := add

	switch *cfg.verb {
	case "add":
		oper = add
	case "del":
		oper = del
	default:
		oper = add
	}

	// Let's grab the configuration
	var rts routes

	// Marshall!
	if _, err := toml.DecodeFile(*cfg.routesfile, &rts); err != nil {
		fmt.Println(err)
		return
	}

	// Init cookie go routine
	req, res := getCookie()

	// Create slice of BgpRouteEntrys (for adds)
	var rtaddslice []*routing.BgpRouteEntry

	// Create a slice of BgpRouteMatches (for deletion)
	var rtdelslice []*routing.BgpRouteMatch

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
			log.Fatalf("Could not read ca certificate: %s", err)
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
	conn, err := grpc.Dial(cfg.hoststring, opts...)

	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer log.Print("Closing connection to ", cfg.hoststring)
	defer conn.Close()

	c := auth.NewLoginClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*cfg.timeout)*time.Second)
	r, err := c.LoginCheck(ctx, &auth.LoginRequest{
		UserName: *cfg.user,
		Password: *cfg.passwd,
		ClientId: *cfg.clientid,
	})

	if err != nil {
		log.Fatalf("Could not connect. Check IP address or domain name: %v", err)
	} else {
		if r.GetResult() {
			log.Printf("Connect to %s: SUCCESS", cfg.hoststring)
		}
	}

	cancel()

	bgpc := routing.NewBgpRouteClient(conn)
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(*cfg.timeout)*time.Second)
	bgprcreply, err := bgpc.BgpRouteInitialize(ctx, &routing.BgpRouteInitializeRequest{})

	if err != nil {
		log.Fatalf("Could not connect to BGP service: %v", err)
	}
	cancel()

	bgpInitReply := routing.BgpRouteInitializeReply_BgpRouteInitializeStatus(bgprcreply.Status)

	if bgpInitReply != routing.BgpRouteInitializeReply_SUCCESS_STATE_REBOUND && bgpInitReply != routing.BgpRouteInitializeReply_SUCCESS {

		log.Fatalf("Error: %s", bgpInitReply.String())

	} else {
		log.Printf("BGP Route API Init: %s", bgpInitReply.String())
	}

	// Create rttname. This doesn't change so moved it from the loops below.
	rttname := &prpd.RouteTableName{Name: "inet.0"}
	rtt := &prpd.RouteTable_RttName{RttName: rttname}
	rtTable := &prpd.RouteTable{RtTableFormat: rtt}

	// Let's build the slice of routes for adding and deletion
	for _, r := range rts.Routes {
		inetPrefix := getInetPrefix(r.Prefix)

		// Build the BgpRouteMatch var for deletion
		bgprm := &routing.BgpRouteMatch{DestPrefix: inetPrefix, DestPrefixLen: r.Length, Table: rtTable, Protocol: routing.RouteProtocol_PROTO_BGP_STATIC, PathCookie: 0}
		// Add the BgpRouteMatch var to the slice (so we can delete "all the routes!"")
		rtdelslice = append(rtdelslice, bgprm)

		// Build next hop table for adds
		for _, n := range r.NextHops {
			nhAddr := &jnxType.IpAddress{AddrFormat: &jnxType.IpAddress_AddrString{AddrString: n}}
			nhAddrSlice := []*jnxType.IpAddress{nhAddr}

			req <- reqCookie
			cookie := <-res

			routeParams := &routing.BgpRouteEntry{
				DestPrefix:       inetPrefix,
				DestPrefixLen:    r.Length,
				Table:            rtTable,
				ProtocolNexthops: nhAddrSlice,
				Protocol:         routing.RouteProtocol_PROTO_BGP_STATIC,
				PathCookie:       cookie,
				RoutePreference:  &routing.BgpAttrib32{Value: rts.Basics.RoutePref},
				LocalPreference:  &routing.BgpAttrib32{Value: rts.Basics.LocalPref},
				Aspath:           &routing.AsPath{AspathString: rts.Basics.AsPathStr},
			}

			rtaddslice = append(rtaddslice, routeParams)
		}
	}

	routeUpdReq := &routing.BgpRouteUpdateRequest{BgpRoutes: rtaddslice}

	if oper == add {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*cfg.timeout)*time.Second)
		defer cancel()

		// This go routine enforces the jetTimeout.
		go func(c context.Context) {
			select {

			case <-c.Done():
				log.Println(c.Err())
			}
		}(ctx)

		result, err := bgpc.BgpRouteAdd(ctx, routeUpdReq)

		if err != nil {
			log.Fatalf("Could not add routes: %v", err)
		}

		log.Printf("Result: %v", result.Status)
	}

	if oper == del {
		removeRequest := &routing.BgpRouteRemoveRequest{OrLonger: false, BgpRoutes: rtdelslice}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*cfg.timeout)*time.Second)
		defer cancel()

		// This go routine enforces the jetTimeout.
		go func(c context.Context) {
			select {

			case <-c.Done():
				log.Println(c.Err())
			}
		}(ctx)

		result, err := bgpc.BgpRouteRemove(ctx, removeRequest)

		if err != nil {
			log.Fatalf("Could not del routes: %v", err)
		}

		log.Printf("Result: %v", result.Status)
	}
}
