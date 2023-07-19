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

// Binary server is an example server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"

	"google.golang.org/grpc"

	pb "google.golang.org/grpc/examples/features/proto/echo"
)

var (
	addrs = map[string]int{":50050": 100, ":50051": 100, ":50052": 100, ":50053": 100, ":50054": 100, ":50055": 100, ":50056": 0, ":50057": 90}
)

type ecServer struct {
	pb.UnimplementedEchoServer
	addr        string
	successRate int
}

func (s *ecServer) UnaryEcho(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	if rand.Intn(100) > s.successRate {
		return nil, errors.New("failing on purpose")
	} else {
		return &pb.EchoResponse{Message: fmt.Sprintf("%s (from %s)", req.Message, s.addr)}, nil
	}
}

func startServer(port string, successRate int) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterEchoServer(s, &ecServer{addr: port, successRate: successRate})
	log.Printf("serving on %s with failures %t\n", port, successRate)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	var wg sync.WaitGroup
	for port, successRate := range addrs {
		wg.Add(1)
		go func(port string, successRate int) {
			defer wg.Done()
			startServer(port, successRate)
		}(port, successRate)
	}
	wg.Wait()
}
