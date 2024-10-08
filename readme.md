# tproxy

English | [简体中文](readme-cn.md) 

<a href="https://www.buymeacoffee.com/kevwan" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

## Why I wrote this tool

When I develop backend services and write [go-zero](https://github.com/zeromicro/go-zero), I often need to monitor the network traffic. For example:
1. monitoring gRPC connections, when to connect and when to reconnect
2. monitoring MySQL connection pools, how many connections and figure out the lifetime policy
3. monitoring any TCP connections on the fly

## Installation

```shell
$ go install github.com/meta-quick/tproxy@latest
```

## Usages

```shell
$ tproxy --help
Usage of tproxy:
  -d duration
    	the delay to relay packets
  -down int
    	Downward speed limit(bytes/second)
  -l string
    	Local address to listen on (default "localhost")
  -p int
    	Local port to listen on, default to pick a random port
  -q	Quiet mode, only prints connection open/close and stats, default false
  -r string
    	Remote address (host:port) to connect
  -s	Enable statistics
  -t string
    	The type of protocol, currently support http2, grpc, redis and mongodb
  -up int
    	Upward speed limit(bytes/second)
  -x  string 
        Relay protocol type, default to tcp, currently support tcp and udp
```

## Examples

### Monitor gRPC connections

```shell
$ tproxy -p 8088 -r localhost:8081 -t grpc -d 100ms
```

- listen on localhost and port 8088
- redirect the traffic to `localhost:8081`
- protocol type to be gRPC
- delay 100ms for each packets

<img width="579" alt="image" src="https://user-images.githubusercontent.com/1918356/181794530-5b25f75f-0c1a-4477-8021-56946903830a.png">

### Monitor MySQL connections

```shell
$ tproxy -p 3307 -r localhost:3306
```

<img width="600" alt="image" src="https://user-images.githubusercontent.com/1918356/173970130-944e4265-8ba6-4d2e-b091-1f6a5de81070.png">

### Check the connection reliability (Retrans rate and RTT)

```shell
$ tproxy -p 3307 -r remotehost:3306 -s -q
```

<img width="548" alt="image" src="https://user-images.githubusercontent.com/1918356/180252614-7cf4d1f9-9ba8-4aa4-a964-6f37cf991749.png">

### Learn the connection pool behaviors

```shell
$ tproxy -p 3307 -r localhost:3306 -q -s
```

<img width="404" alt="image" src="https://user-images.githubusercontent.com/1918356/236633144-9136e415-5763-4051-8c59-78ac363229ac.png">

## Give a Star! ⭐

If you like or are using this project, please give it a **star**. Thanks!
