## jRoutes BGP Example

This project replicates what Marcel Wiget did with the jroutes Python JET off-box example which can be found [here](https://github.com/mwiget/jet-bgp-static-routes/blob/master/jroutes_bgp.py).

Marcel's original Python code injects BGP static routes into Junos with multiple next-hops using the BGP JET API through gRPC calls. This code does exactly the same and exercises some Go to do so! The code in some ways is over engineered and really is nothing more than a glorified script rather than an application. It should give any newcomers to `Go` a good introduction.

*Please note, any line prepended with a hash (#) represents a comment or feedback from a bash or CLI command.*

In case you do not want to build the binaries or do not have the knowledge, I've included three pre-built versions in this directory:

```bash
bgp_static_routes-junos-32-0.1  = Compiled for 32 bit FreeBSD (runs on Junos)
bgp_static_routes-linux-64-0.1  = Compiled for any 64 bit Linux
bgp_static_routes-osx-0.1       = Compiled for OSX
```

## Basics

Please make sure you followed the README in the repository root directory (one up from here) for setting up the environment and tool chain.

Also, specifically for this demo, please make sure the following configuration is in place on your MX/vMX.
__WARNING: DO NOT DO THIS ON A LIVE ENVIRONMENT. WE CHANGE THE AUTONOMOUS SYSTEM NUMBER AND ENABLE NEW SERVICES. PLEASE BE CAREFUL. I AM NOT RESPONSIBLE FOR YOU!__

```bash
set protocols bgp group internal type internal
set protocols bgp group internal family inet unicast add-path send path-count 6
set protocols bgp group internal allow 0.0.0.0/0
set routing-options autonomous-system 64512
set routing-options programmable-rpd purge-timeout 20
```

It's prudent to mention that it is always preferred to run these applications with TLS instead of clear text. Whilst it's possible to run the app clear text (leave off the -certdir argument), I do not promote this! Below is the Junos config snippet required to setup mutual authentication with SSL for this example (assuming basic knowledge of creating the PKI tooling and the CA details on Junos).

```bash
set system services extension-service request-response grpc ssl local-certificate vmx01.domain
set system services extension-service request-response grpc ssl mutual-authentication certificate-authority CA
set system services extension-service request-response grpc ssl mutual-authentication client-certificate-request require-certificate
```

## Build

Change directory in to the demo directory and copy the example config file, modify its contents to reflect your settings and save. Follow the steps below!

```bash
cd github.com/arsonistgopher/go-jet-demo-apps/bgp_static_routes
cp example_routes.toml routes.toml
vim config.toml  
```

Using VIM (the text editor) will allow you to change the contents of the configuration file.

```bash
[basics]
localPref  = 200
routePref  = 10
asPathStr  = ""
originator = "10.255.255.3"
cluster    = "10.255.255.7"

[[route]]
prefix = "10.123.0.0"
length = 24
nexthops = ["10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"]

[[route]]
prefix = "10.123.1.0"
length = 24
nexthops = ["10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"]
```

Once you're done editing, issue `esc` and `:wq!` then hit return. The file's modified contents will be saved.

Now brave solider, you can build the demo!

```bash
go build
```

Even better, you can now run your demo app with the config file you've just modified! The `-verb` switch below can only be `add` or `del` for adding and removing prefixes respectively via JET. I'm choosing to add.
Ensure that you have a user account created on Junos with the correct privileges to make modifications.

```bash
./bgp_static_routes -certdir CLIENTCERT -host vmx01 -routesfile routes.toml -user jet -verb add
```

The output if everything goes well?

```bash
2018/05/08 19:27:43 --------------------------------------
2018/05/08 19:27:43 Junos JET BGP-Static Route Test Client
2018/05/08 19:27:43 --------------------------------------
2018/05/08 19:27:43 Run the app with -h for options

2018/05/08 19:27:43 Enter Password:
2018/05/08 19:27:45 Connect to vmx01.domain:32767: SUCCESS
2018/05/08 19:27:46 BGP Route API Init: SUCCESS
2018/05/08 19:27:46 Result: SUCCESS
2018/05/08 19:27:46 Closing connection to vmx01.domain:32767
```

And the output on the Juniper vMX?

```bash
root@vmx01> show route
*snip*
10.123.0.0/24      *[BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.1 via ge-0/0/0.0
                    [BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.2 via ge-0/0/0.0
                    [BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.3 via ge-0/0/0.0
                    [BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.4 via ge-0/0/0.0
10.123.1.0/24      *[BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.1 via ge-0/0/0.0
                    [BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.2 via ge-0/0/0.0
                    [BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.3 via ge-0/0/0.0
                    [BGP-Static/10/-201] 00:00:02, metric2 0
                    > to 10.0.0.4 via ge-0/0/0.
*snip*
```

A la working demo.
