## Demo Golang JET Apps

This repository contains demo Junos JET applications written in Golang.

Each sub-directory contains individual READMEs with instructions on how to build each project.

## Building the demos

In order to run these demos from zero previous Go experience, we need to install a number of things and also obtain some files.

1. Collect IDL files for your version of Junos from the Juniper download site. This link will take you to the download site for the IDL files. Be sure to select the right version of IDL files! [https://www.juniper.net/support/downloads/?p=jet](https://www.juniper.net/support/downloads/?p=jet)

2. Next, install the latest version of Go. At the time of writing it is 1.10.1 and that is the validated working version for this demo. The download link is here: [https://golang.org/dl/](https://golang.org/dl/).

3. Once you've installed Golang on your system, ensure that the $GOROOT and $GOBIN environmental variables are set. You can follow this guide to achieve this step: [https://golang.org/doc/install](https://golang.org/doc/install).

4. Install the `protoc` GRPC compiler from a binary. As we're going to be compiling the IDL (`.proto`) files for Go, we're also going to need to install the `Go` plugin. Here [https://github.com/google/protobuf/releases](https://github.com/google/protobuf/releases) is where to download the binary. For this demo, use version 3.5.1 for your operating system. Here is how to get the `Go` plugin for `protoc`.

```bash
go get -u github.com/golang/protobuf/protoc-gen-go
```

5. Next, clone this repository and enter the directory.

```bash
git clone https://github.com/DavidJohnGee/go-jet-demo-app.git
cd go-jet-demo-app
```

6. Create a directory called `proto` in this directory (command below). Also, copy the IDL files you previously downloaded to the `proto` directory and extract them. Note, my storage location for IDL files will not be the same as yours. Ensure you copy using the correct path and do not blindly copy the below.

```bash
mkdir proto
cp ~/Documents/JET/jet-idl-17.4R1.16.tar.gz ./proto
tar xzvf ./proto/jet-idl-17.4R1.16.tar.gz
```

Next, we need to compile four different proto files. These are: `jnx_addr.proto`, `authentication_service.proto`, `prpd_common.proto` and `bgp_route_service.proto`.

```bash
 protoc -I proto proto/jnx_addr.proto --go_out=plugins=grpc:proto
 protoc -I proto proto/authentication_service.proto --go_out=plugins=grpc:proto
 protoc -I proto proto/prpd_common.proto --go_out=plugins=grpc:proto
 protoc -I proto proto/bgp_route_service.proto --go_out=plugins=grpc:proto
```

From the compilation step above, the output falls now look like:

```bash
authentication_service.pb.go
bgp_route_service.pb.go
jnx_addr.pb.go
prpd_common.pb.go
```

We're not done quite just yet however. Next, we need to ensure that the `prpd_common.pb.go` and `bgp_route_service.pb.go` are accessible in their own package space. From a generated code perspective, when one automagically generated set of code tries to import code from another automagically generated file with the same package name, in the file that is importing, the package to be imported will have a number `1` appended to the end to make it unique. In Go, given a fictious example, if you have `Package Bob`, one cannot simply import `Bob`; because Bob is Bob. This makes for pretty ugly code and despite it saying very clearly "DO NOT EDIT" in the automagically generated code, for reasons of relliability, determinism and engineering "bad luck away" I am going to do exactly that.

The two excerpts below are excerpts from the automagically generated Go code. Note the Go files import `.` and note the `routing1` import. Not good and not clear.

This is the `bgp_route_service.pb.go` excerpt.

```bash
package routing

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import jnxBase "."
import routing1 "."
```

This is the `prpd_common.pb.go` excerpt:

```bash
package routing

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import jnxBase "."
```

Three things have to change here, the `jnxBase` import, change of `routing1` to `prpd` import and the `routing` package name in prpd_common to `prpd`. Here's what the excerpt looks like now after modifying the top of the `bgp_route_service.pb.go` file.

```bash
package routing

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import jnxBase "github.com/davidjohngee/go-jet-demo-app/proto/jnxBase"
import prpd "github.com/davidjohngee/go-jet-demo-app/proto/prpd"
```

...and the `prpd_common.pb.go` code changes.

```bash
package prpd

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import jnxBase "github.com/davidjohngee/go-jet-demo-app/proto/jnxBase"
```

Next step, within the `proto` directory, we need to create subdirectories for each `Go` proto file and copy the file in to them.

```bash
cd proto
mkdir -p {authentication,jnxBase,routing,prpd}
mv authentication_service.pb.go ./authentication
mv bgp_route_service.pb.go ./routing
mv jnx_addr.pb.go ./jnxBase
mv prpd_common.pb.go ./prpd
```

Last challenge, within the `bgp_route_service.pb.go` file, replace every instance of `routing1` with `prpd`.

## bgp_static_routes directory

This directory contains the first demo that uses the code generated from the `.proto` files.

If you want to be lazy, use [this link](https://github.com/DavidJohnGee/go-jet-demo-app/tree/master/bgp_static_routes) to go to the demo directory.
