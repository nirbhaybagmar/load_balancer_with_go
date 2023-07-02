package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

type loadBalancer struct {
	port            string
	servers         []Server
	roundRobinCount int
}

func newSimpleServer(addr string) *simpleServer {
	serveUrl, _ := url.Parse(addr)
	// handleError(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serveUrl),
	}
}

func newLoadBalancer(port string, servers *[]Server) *loadBalancer {
	return &loadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         *servers,
	}
}

func (lb *loadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]

	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++

	return server

}

func (lb *loadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to addr %s\n", targetServer.Address())
	targetServer.Serve(rw, r)
}

func (s simpleServer) Address() string {
	return s.addr
}

func (s simpleServer) IsAlive() bool {
	return true
}

func (s simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

func handleError(err error) {
	fmt.Printf("Error: %+v\n", err)
	os.Exit(1)
}

func main() {

	// Create three servers 
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.google.com"),
	}

	// Create a load balancers and assign the servers 
	lb := newLoadBalancer("8000", &servers)
	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("Serving Requests at localhost:%s \n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)

}
