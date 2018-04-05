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
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/BurntSushi/toml"
	auth "github.com/davidjohngee/go-jet-demo-app/proto/authentication"
	jnxBase "github.com/davidjohngee/go-jet-demo-app/proto/jnxBase"
	prpd "github.com/davidjohngee/go-jet-demo-app/proto/prpd"
	routing "github.com/davidjohngee/go-jet-demo-app/proto/routing"

	"google.golang.org/grpc"
)

const (
	// reqCookie is a const for requesting a unique cookie
	reqCookie = uint8(0)
	add       = 0
	del       = 1
)

// custom struct route type for loading our configuration based routes.
type route struct {
	Prefix   string   `toml:"prefix"`
	Length   uint32   `toml:"length"`
	NextHops []string `toml:"nexthops"`
}

// custom struct route type for loading our configuration based routes.
type core struct {
	Address    string `toml:"address"`
	Username   string `toml:"username"`
	Passwd     string `toml:"passwd"`
	Clientid   string `toml:"clientid"`
	JetTimeout int    `toml:"jetTimeout"`

	LocalPref  uint32 `toml:"localPref"`
	RoutePref  uint32 `toml:"routePref"`
	AsPathStr  string `toml:"asPathStr"`
	Originator string `toml:"originator"`
	Cluster    string `toml:"cluster"`
}

// TOML based config struct for loading from a configuration file.
type tomlConfig struct {
	Core   core
	Routes []route `toml:"route"`
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
	inetAddr := &jnxBase.IpAddress{AddrFormat: &jnxBase.IpAddress_AddrString{AddrString: s}}
	inetPrefixInet := &prpd.RoutePrefix_Inet{Inet: inetAddr}
	inetPrefix := &prpd.RoutePrefix{RoutePrefixAf: inetPrefixInet}
	return inetPrefix
}

func main() {
	// Dirty flag for quick testing of verb
	var verb = flag.String("verb", "add", "Verb is 'add' or 'del'")
	flag.Parse()

	// Oper is the operational verb: add/del routes
	oper := add

	switch *verb {
	case "add":
		oper = add
	case "del":
		oper = del
	default:
		oper = add
	}

	// Let's grab the configuration
	var config tomlConfig

	// Marshall!
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		fmt.Println(err)
		return
	}

	// Init cookie go routine
	req, res := getCookie()

	// Create slice of BgpRouteEntrys (for adds)
	var rtaddslice []*routing.BgpRouteEntry

	// Create a slice of BgpRouteMatches (for deletion)
	var rtdelslice []*routing.BgpRouteMatch

	// Set up a connection to the server.
	conn, err := grpc.Dial(config.Core.Address, grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer log.Print("Closing connection to ", config.Core.Address)
	defer conn.Close()

	c := auth.NewLoginClient(conn)

	r, err := c.LoginCheck(context.Background(), &auth.LoginRequest{
		UserName: config.Core.Username,
		Password: config.Core.Passwd,
		ClientId: config.Core.Clientid,
	})

	if err != nil {
		log.Fatalf("Could not connect. Check IP address or domain name: %v", err)
	} else {
		if r.GetResult() {
			log.Printf("Connect to %s: SUCCESS", config.Core.Address)
		}
	}

	bgpc := routing.NewBgpRouteClient(conn)

	bgprcreply, err := bgpc.BgpRouteInitialize(context.Background(), &routing.BgpRouteInitializeRequest{})

	if err != nil {
		log.Fatalf("Could not connect to BGP service: %v", err)
	}

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
	for _, r := range config.Routes {
		inetPrefix := getInetPrefix(r.Prefix)

		// Build the BgpRouteMatch var for deletion
		bgprm := &routing.BgpRouteMatch{DestPrefix: inetPrefix, DestPrefixLen: r.Length, Table: rtTable, Protocol: routing.RouteProtocol_PROTO_BGP_STATIC, PathCookie: 0}
		// Add the BgpRouteMatch var to the slice (so we can delete "all the routes!"")
		rtdelslice = append(rtdelslice, bgprm)

		// Build next hop table for adds
		for _, n := range r.NextHops {
			nhAddr := &jnxBase.IpAddress{AddrFormat: &jnxBase.IpAddress_AddrString{AddrString: n}}
			nhAddrSlice := []*jnxBase.IpAddress{nhAddr}

			req <- reqCookie
			cookie := <-res

			routeParams := &routing.BgpRouteEntry{
				DestPrefix:       inetPrefix,
				DestPrefixLen:    r.Length,
				Table:            rtTable,
				ProtocolNexthops: nhAddrSlice,
				Protocol:         routing.RouteProtocol_PROTO_BGP_STATIC,
				PathCookie:       cookie,
				RoutePreference:  &routing.BgpAttrib32{Value: config.Core.RoutePref},
				LocalPreference:  &routing.BgpAttrib32{Value: config.Core.LocalPref},
				Aspath:           &routing.AsPath{AspathString: config.Core.AsPathStr},
			}

			rtaddslice = append(rtaddslice, routeParams)
		}
	}

	routeUpdReq := &routing.BgpRouteUpdateRequest{BgpRoutes: rtaddslice}

	if oper == add {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Core.JetTimeout)*time.Second)
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

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Core.JetTimeout)*time.Second)
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
