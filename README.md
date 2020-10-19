A simple POC of an SSH-based web terminal.

`/client`:
- emulates a dummy device
- runs an actual ssh server
- and a proxy to expose it via a websocket
- enforces public key auth

`/deviceconnect`:
- backend service handling the remote terminal <-> SSH <-> device websocket communication
- exposes a simple terminal (xterm.js) ui

## Running
```
make build
make up
navigate to localhost:8080
```
You'll get an SSH session in a terminal immediately.
It's just one user (you) and a single device; both ends are connected immediately when the setup starts.

## How it works
The idea is to:
- implement the remote terminal using actual ssh
- but also: use only websockets for ssh connectivity - so no ssh tunnels, extra ports, jumpboxes, etc.

The basic assumption is that:
- there's 1 ws connection from the user (specifically, user's web-based terminal)
- and 1 ws connection from the device

To enforce ssh - the **device** actually contains a running ssh daemon. 
It's build on gliderlabs/ssh and simply listens in-process for incoming sessions. 
It listens on the standard port, the port is not exposed outside however.

Side by side, the device runs a proxy (also in-process), which is connected:
- via WS to the server
- via a tcp port to the above ssh server

The proxy starts a pipeline pumping data between both ends, marshalling as necessary. WS messages are streamed as bytes, conversely - bytes from the tcp connection are sent as WS messages.

Whatever sends valid ssh data stream over this websocket can talk to the device in a secure manner.

On the **server** side, deviceconnect accepts the user's WS connection and is ready to accept his input.

At this point we need some kind of ssh agent on the server which will actually init the ssh session and start pumping data.

Therefore, on WS upgrade/terminal session initiation:
- an ad hoc ssh client is created (starts `/bin/bash`)
- it can't speak websockets straight to the device, so
- an ad hoc tcp <-> ws adapter is also started on a semi random port
- the client speaks to the adapter, which repacks the data stream in to WS frames and back 
- the user's websocket is coupled with the ssh client's std streams via another data pump

To sum up, the data pipeline from the user's perspective:
- terminal input is captured at the websocket
- goes into the ssh client via stdin (like with a regular ssh cli)
- as ssh data bytestream, goes to the adapter
- translated to WS frames, goes to the device
- the device proxy unpacks WS frames to a byte stream
- and funnels it into :22

From the device's perspective:
- sshd sends a stream of data via tcp to the proxy
- each chunk is packed into WS frames
- the adapter on the server unpacks WS frames into bytes
- feeds them into the ssh client's tcp end
- client simply speaks on its stdout (like a regular ssh client would)
- the stdout stream is packet into WS frames and sent to the terminal for display

As a result, a real ssh session with auth and encryption happens end-to-end over websocket connections.

## Misc
- just a single user's key (id_rsa) is set up; we could have one per actual user, one per tenant, one per the whole backend, matter of implementation
- the code is quick and dirty - there are sometimes stability issues because of poor practices (no real resource mgmt, defer Close()s, contexts, timeouts...) 
