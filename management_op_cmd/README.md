## junos-jet-op-cmd-testclient

This simple application tests the Junos JET OpCommand gRPC API. In super simple terms, the script consumes the JET IDL (`.proto`) and calls the `ExecuteOpCommand` service via gRPC.

The application itself is available in three forms pre-compiled and ready to use. This is handy if you're not familiar with the build process for `.proto` based applications. Read more on this [here](https://github.com/arsonistgopher/junos-jet-demo-apps) if you want to build this from source!

Please note, this application requires that PKI has been dealt with. This in turn means, Junos has a CA certificate, a local certificate and a local key. In addition, the application also requires it's own certificate and key. The result is secure, mutually authenticated TLS between this application and Junos.

* May 2018 - I'm working on a blog post which describes configuring a full PKI example with Junos. Patience please! *

## To use the applications

__1__  Clone or download this repository.

__2__  Choose the application for your operating system.

```bash
management_op_cmd-osx-0.1       = an application that can be executed on any OSX based OS
management_op_cmd-linux-64-0.1  = an application that can be executed on any Linux 64 bit system
management_op_cmd-junos-32-0.1  = an application that can be packaged for and executed on 32 bit Junos
```

__3__  Execute the binary with command line arguments. These arguments are available to inspect with `-h`.

```bash
./management_op_cmd-osx-0.1 -h
2018/05/04 14:46:51 ------------------------------
2018/05/04 14:46:51 Junos JET OpCommand Test Tool
2018/05/04 14:46:51 ------------------------------
2018/05/04 14:46:51 Run the app with -h for options

Usage of ./management_op_cmd-osx-0.1:
  -certdir string
    	Directory with client.crt, client.key, CA.crt
  -cid string
    	Client ID for session (default "42")
  -command string
    	Operational command (default "show version")
  -format string
    	XML or JSON (default "xml")
  -host string
    	Hostname or IP Address (default "127.0.0.1")
  -passwd string
    	Password for Junos host. Note, not mandatory
  -port string
    	Port that the grpc server is listening on. (default "32767")
  -timeout int
    	Timeout in seconds for JET (default 10)
  -user string
    	Username for authentication (default "jet")
```

Here is an example run on a system configured to accept clear-text gRPC.

```bash
./management_op_cmd-osx-0.1 -certdir CLIENTCERT -command "show system information" -host vmx01.corepipe.co.uk
2018/05/04 14:54:06 ------------------------------
2018/05/04 14:54:06 Junos JET OpCommand Test Tool
2018/05/04 14:54:06 ------------------------------
2018/05/04 14:54:06 Run the app with -h for options

2018/05/04 14:54:06 Enter Password:
2018/05/04 14:54:08 Unrecognised format type. Defaulting to XML
2018/05/04 14:54:08 Connect to vmx01.corepipe.co.uk successful

---Data---

<system-information>
<hardware-model>vmx</hardware-model>
<os-name>junos</os-name>
<os-version>18.1R1.9</os-version>
<serial-number>VM5AE9D436D2</serial-number>
<host-name>vmx01</host-name>
</system-information>
```

Note, the comment `Unrecognised format type`. This means no argument was passed in. Here is the same inputs but this time with a request for `JSON` based output.

```bash
./management_op_cmd-osx-0.1 -certdir CLIENTCERT -command "show system information" -host vmx01.corepipe.co.uk -format json
2018/05/04 14:55:49 ------------------------------
2018/05/04 14:55:49 Junos JET OpCommand Test Tool
2018/05/04 14:55:49 ------------------------------
2018/05/04 14:55:49 Run the app with -h for options

2018/05/04 14:55:49 Enter Password:
2018/05/04 14:55:52 Connect to vmx01.corepipe.co.uk successful

---Data---

{
    "system-information" : [
    {
        "hardware-model" : [
        {
            "data" : "vmx"
        }
        ],
        "os-name" : [
        {
            "data" : "junos"
        }
        ],
        "os-version" : [
        {
            "data" : "18.1R1.9"
        }
        ],
        "serial-number" : [
        {
            "data" : "VM5AE9D436D2"
        }
        ],
        "host-name" : [
        {
            "data" : "vmx01"
        }
        ]
    }
    ]
}
```

Enjoy.
