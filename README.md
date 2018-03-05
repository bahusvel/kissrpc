# KISS (Keep It Simple, Stupid) RPC (Remote Procedure Call)
The simplest possible RPC library for Go. Having been awfully frustrated by stupid requirements the [native rpc package](https://golang.org/pkg/net/rpc/), lack of type safety of others, and focus on irrelevant things like compression and encryption I decided to make this.

This is the most flexible RPC library for Go, which implements encryption and compression in whichever algorithm you like by not implementing it. KissRPC uses net.Conn as the base interface for all of its communication, so as long as you are able to provide it something that looks like net.Conn it will be happy.

KissRPC implements type safe RPC by declaring services as structs of functions, which allow to avoid dodgy casting from interface{} and back, is visible to autocomplete and looks completely natural. (Non type safe is also supported, and the casting is done automatically).

KissRPC enforces no format on the methods it can export, it supports everything that go does (except for channels), because it uses [gob](https://golang.org/pkg/encoding/gob/) for encoding and reflection for automatic casting. (Yes reflection is slow, and KissRPC lets you avoid it if you want, but it primarly focuses on ease of use, and blindly sacrificing everything for performance.)

If you are frustrated with other RPC libraries, try this one. The documentation is [here](https://golang.org/pkg/net/rpc/) and the code is small so hack away!
