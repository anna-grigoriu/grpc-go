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

var addrs = []string{"localhost:50051", "localhost:50052", "localhost:50053", "localhost:50054"}

func callUnaryEcho(c ecpb.EchoClient, message string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	hwc := ecpb.NewEchoClient(cc)
	done := false
	successes := map[string]int{}
	failures := 0
	ch := time.After(time.Minute)
	for !done {
		select {
		case <-ch:
			done = true
		case <-time.After(500 * time.Millisecond):
			message, err := callUnaryEcho(hwc, "this is examples/load_balancing")
			if err != nil {
				failures++
			} else {
				successes[message]++
			}
		}
	}
	log.Printf("In the first minute, we had %d failures and %d healthy servers", failures, len(successes))
	done = false
	successes = map[string]int{}
	failures = 0
	for !done {
		select {
		case <-ctx.Done():
			done = true
		case <-time.After(500 * time.Millisecond):
			message, err := callUnaryEcho(hwc, "this is examples/load_balancing")
			if err != nil {
				failures++
			} else {
				successes[message]++
			}
		}
	}
	log.Printf("In the second minute, we had %d failures and %d healthy servers", failures, len(successes))
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
				"interval": 10000000000,
				"baseEjectionTime": 30000000000,
				"maxEjectionTime": 300000000000,
				"maxEjectionPercent": 10,
				"successRateEjection": {
					"stdevFactor": 1500,
					"enforcementPercentage": 100,
					"minimumHosts": 3,
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
