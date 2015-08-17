package main

import "encoding/json"
import "flag"
import "log"
import "time"
import "github.com/AdRoll/goamz/aws"
import "github.com/AdRoll/goamz/s3"
import "github.com/mikroio/tcp-forward-proxy/discovery"
import "github.com/mikroio/tcp-forward-proxy/proxy"

var region *string = flag.String("r", "", "aws region")
var table *string = flag.String("t", "uio-jrydberg-default", "service disovery table")

type RoutingEntry struct {
  Name string `json:"name"`
  ListenPort int `json:"listen_port"`
  TargetService string `json:"target_service"`
}

type RoutingProxy struct {
  service string
  serviceDiscovery *discovery.Discovery
  serviceProxy *proxy.Proxy
}

var proxies map[int]*RoutingProxy = make(map[int]*RoutingProxy)

var proxyConfig proxy.Config = proxy.Config{
  MaxPoolSize: 30,
  ConnectTimeout: 10,
}

func updateProxies(entries []RoutingEntry) {
  // FIXME: support routes that has been gone from the routing table.

  for _, entry := range entries {
    routingProxy, ok := proxies[entry.ListenPort]
    if ok {
      if routingProxy.service == entry.TargetService {
        continue
      }

      delete(proxies, entry.ListenPort)
      routingProxy.serviceProxy.Close()
      routingProxy.serviceDiscovery.Close()
    }

    serviceDiscovery := discovery.New(entry.TargetService,
                                      *region, *table)
    serviceDiscovery.Start()

    serviceProxy := proxy.New(serviceDiscovery, proxyConfig)
    err := serviceProxy.Listen(entry.ListenPort)
    if err != nil {
      log.Print("cannot start listen on port", entry.ListenPort)
      continue
    }

    go serviceProxy.Accept()

    proxies[entry.ListenPort] = &RoutingProxy{
      service: entry.TargetService,
      serviceDiscovery: serviceDiscovery,
      serviceProxy: serviceProxy,
    }
  }
}

func main() {
  flag.Parse()

  if *region == "" {
    *region = aws.InstanceRegion()
  }

  auth, err := aws.GetAuth("", "", "", time.Now())
  if err != nil {
    log.Panic(err)
  }

  s3service := s3.New(auth, aws.GetRegion(*region))
  bucket := s3service.Bucket(flag.Arg(0))

  for {
    var entries []RoutingEntry

    data, err := bucket.Get("/routing-table.json")
    if err == nil {
      err = json.Unmarshal(data, &entries)
      if err == nil {
        updateProxies(entries)
      }
    } else {
      log.Print("no get routing table", err)
    }

    time.Sleep(time.Second * 10)
  }
}
