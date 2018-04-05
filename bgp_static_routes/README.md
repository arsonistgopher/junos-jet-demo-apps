## jRoutes BGP Example

This project replicates what Marcel Wiget did with the jroutes Python JET off-box example which can be found [here](https://github.com/mwiget/jet-bgp-static-routes/blob/master/jroutes_bgp.py).

Marcel's original Python code injects BGP static routes into Junos with multiple next-hops using the BGP JET API through gRPC calls. This code does exactly the same and exercises some Go to do so! The code in some ways is over engineered and really is nothing more than a glorified script rather than an application. It should give any newcomers to `Go` a good introduction.

*Please note, any line prepended with a hash (#) represents a comment or feedback from a bash or CLI command.*

## Basics

Please make sure you followed the README in the repository root directory (one up from here) for setting up the environment and tool chain.

Also, specifically for this demo, please make sure the following configuration is in place on your MX/vMX.
__WARNING: DO NOT DO THIS ON A LIVE ENVIRONMENT. WE OPEN UP THE DEVICE TO THE WORLD AND CHANGE THE AUTONOMOUS SYSTEM NUMBER. PLEASE BE CAREFUL. I AM NOT RESPONSIBLE FOR YOU!__

```bash
set system services extension-service request-response grpc clear-text port 50051
set system services extension-service request-response grpc skip-authentication
set protocols bgp group internal type internal
set protocols bgp group internal family inet unicast add-path send path-count 6
set protocols bgp group internal allow 0.0.0.0/0
set routing-options autonomous-system 64512
set routing-options programmable-rpd purge-timeout 20
```

## Build

Change directory in to the demo directory and copy the example config file, modify its contents to reflect your settings and save. Follow the steps below!

```bash
cd github.com/davidjohngee/go-jet-demo-app/bgp_static_routes
cp example_config.toml config.toml
vim config.toml  
```

Using VIM (the text editor) will allow you to change the contents of the configuration file. Please set fields correctly and save. I've included a basic example for my lab below. This isn't accessible by you, but it serves a purpose of showing you the configuration fields. Also, please make sure your user account exists with the password and user class set appropriately.

```bash
[core]
address    = "192.168.100.100:50051"
username   = "jet"
passwd     = "Passw0rd"
clientid   = "42"
jetTimeout = 10

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

```bash
./bgp_static_routes -verb add
```

The output if everything goes well?

```bash
2018/04/05 19:41:08 Connect to 192.168.100.100:50051: SUCCESS
2018/04/05 19:41:08 BGP Route API Init: SUCCESS
2018/04/05 19:41:08 Result: SUCCESS
2018/04/05 19:41:08 Closing connection to 192.168.100.100:50051
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

