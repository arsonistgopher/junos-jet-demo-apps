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
git clone https://github.com/arsonistgopher/junos-jet-demo-apps.git
cd go-jet-demo-app
```

6. Copy the `jet-idl` file in to this demo directory and extract the files. In my example I'm copying them from my Documents directory, which is probably not where your IDL tar file is located.

```bash
cp ~/Documents/JET/jet-idl-17.4R1.16.tar.gz ./
tar xzvf ./proto/jet-idl-17.4R1.16.tar.gz
```

Next, we need to compile four different proto files and put them in the right place for our Go code! For this, I've cheated a little by creating a `bash` script which compiles the required files.

```bash
source runme.sh
```

## Demo Applications

Pre-compiled versions exist, so don't worry if you're not an expert with `Go` or the `protoc` tool.
*Warning: Pre-compiled applications use the 18.1 version of IDLs, as such they might not work as expected with older systems*

__bgp_static_routes directory__

This application inserts BGP-Static routes in to Junos via the JET API over gRPC using the compiled IDL files in Go! 


__management_op_cmd directory__

This application executes an operational command via the JET management API over gRPC using the compiled IDL files in Go!

__mqtt_bridge__

This application bridges messages received on a configurable topic on the MQTT broker to `eventd`, meaning MQTT messages can be used to trigger op scripts or other things on Junos or any system consuming Junos event messages.
