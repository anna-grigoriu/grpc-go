/*
 *
 * Copyright 2018 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Binary client is an example client.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	ecpb "google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/resolver"
	_ "google.golang.org/grpc/xds" //nolint // using outlier detection requires this import.
)

const (
	exampleScheme      = "example"
	exampleServiceName = "lb.example.grpc.io"
)

var addrs = []string{"localhost:50050", "localhost:50051", "localhost:50052", "localhost:50053", "localhost:50054", "localhost:50055", "localhost:50056", "localhost:50057"}

func callUnaryEcho(c ecpb.EchoClient, message string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.UnaryEcho(ctx, &ecpb.EchoRequest{Message: message})
	if err != nil {
		log.Printf("could not greet: %v", err)
		return "", err
	}
	fmt.Println(r.Message)
	return r.Message, nil
}

func makeRPCs(cc *grpc.ClientConn, n int) {
	hwc := ecpb.NewEchoClient(cc)
	done := false
	ch := time.After(5 * time.Minute)
	for !done {
		select {
		case <-ch:
			done = true
		case <-time.After(100 * time.Millisecond):
			callUnaryEcho(hwc, "this is examples/load_balancing")
		}
	}
}

func main() {
	// Make another ClientConn with OutlierDetection round_robin policy.
	roundrobinConn, err := grpc.Dial(
		fmt.Sprintf("%s:///%s", exampleScheme, exampleServiceName),
		grpc.WithDefaultServiceConfig(`
		{
		  "loadBalancingConfig": [
			{
			  "outlier_detection_experimental": {
				"interval": "10s",
				"baseEjectionTime": "30s",
				"maxEjectionTime": "300s",
				"maxEjectionPercent": 30,
				"failurePercentageEjection": {
					"threshold": 85,
					"enforcementPercentage": 100,
					"minimumHosts": 3,
					"requestVolume": 5
				},
				"successRateEjection": {
					"stdevFactor" : 1900,
					"enforcementPercentage": 100,
					"minimumHosts" : 6,
					"requestVolume": 5
				},
				"childPolicy": [{"round_robin": {}}]
			  }
			}
		  ]
		}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer roundrobinConn.Close()

	fmt.Println("--- calling helloworld.Greeter/SayHello with OutlierDetection RoundRobin ---")
	makeRPCs(roundrobinConn, 20)
}

// Following is an example name resolver implementation. Read the name
// resolution example to learn more about it.

type exampleResolverBuilder struct{}

func (*exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &exampleResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			exampleServiceName: addrs,
		},
	}
	r.start()
	return r, nil
}
func (*exampleResolverBuilder) Scheme() string { return exampleScheme }

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *exampleResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint()]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*exampleResolver) Close()                                  {}

func init() {
	resolver.Register(&exampleResolverBuilder{})
}
