## OpCmd Tester App

This simple application tests the Junos JET OpCommand gRPC API. In super simple terms, the script consumes the JET IDL (`.proto`) and calls the `ExecuteOpCommand` service via gRPC.

The application itself is available in three forms pre-compiled and ready to use. This is handy if you're not familiar with the build process for `.proto` based applications. Read more on this [here](https://github.com/DavidJohnGee/go-jet-demo-app) if you want to build this from source!

## To use the applications

__1__  Clone or download this repository.

__2__  Choose the application for your operating system.

```bash
opcmdjunos32 = an application that can be packaged for Junos
opcmdlinux64 = an application that can be executed on any Linux 64 bit OSes
opcmdosx     = an application that can be executed on any OSX based OS
```

__3__  Execute the binary with command line arguments. These arguments are available to inspect with `-h`.

```bash
./opcmdosx -h

Usage of ./opcmdosx:
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
    	Port that the grpc server is listening on (default "50051")
  -timeout int
    	Timeout in seconds for JET (default 10)
  -user string
    	Username for authentication (default "jet")
```

Here is an example run on a system configured to accept clear-text gRPC.

```bash
./opcmdosx -command "show system info" -host vmx01 -port 50051 -user jet
2018/05/02 21:29:43 ------------------------------
2018/05/02 21:29:43 Junos JET OpCommand Test Tool
2018/05/02 21:29:43 ------------------------------
2018/05/02 21:29:43 Run the app with -h for options

2018/05/02 21:29:43 Enter Password:
2018/05/02 21:29:45 Unrecognised format type. Defaulting to XML
2018/05/02 21:29:45 Connect to vmx01 successful

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
./opcmdosx -command "show system info" -host vmx01 -port 50051 -user jet -format JSON
2018/05/02 21:31:00 ------------------------------
2018/05/02 21:31:00 Junos JET OpCommand Test Tool
2018/05/02 21:31:00 ------------------------------
2018/05/02 21:31:00 Run the app with -h for options

2018/05/02 21:31:00 Enter Password:
2018/05/02 21:31:02 Connect to vmx01 successful

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

For the gRPC JET APIs to be functional, you are required to install the "network-agent" package and the "openconfig" packages on Junos.

Once these are installed, some basic configuration is required to enable clear-text gRPC. I do not recommend this for anything other than testing. Always use TLS for production systems and mutual TLS preferably.

```bash
set system services extension-service request-response grpc clear-text port 50051
set system services extension-service request-response grpc skip-authentication
```

Also ensure that you have a test user account setup and know the password.

Enjoy.
