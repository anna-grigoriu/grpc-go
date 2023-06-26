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
	"net"
	"sync"

	"google.golang.org/grpc"

	pb "google.golang.org/grpc/examples/features/proto/echo"
)

var (
	addrs = map[string]bool{":50051": true, ":50052": false, ":50053": false, ":50054": false}
)

type ecServer struct {
	pb.UnimplementedEchoServer
	addr      string
	maybeFail bool
}

func (s *ecServer) UnaryEcho(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	if s.maybeFail {
		return nil, errors.New("failing on purpose")
	} else {
		return &pb.EchoResponse{Message: fmt.Sprintf("%s (from %s)", req.Message, s.addr)}, nil
	}
}

func startServer(port string, fails bool) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterEchoServer(s, &ecServer{addr: port, maybeFail: fails})
	log.Printf("serving on %s with failures %t\n", port, fails)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	var wg sync.WaitGroup
	for port, fails := range addrs {
		wg.Add(1)
		go func(port string, fails bool) {
			defer wg.Done()
			startServer(port, fails)
		}(port, fails)
	}
	wg.Wait()
}
