# Raft gRPC definitions

This package contains the definition for protobufs used in the raft protocol in this repository. the *.pb.go files are compiled using the rpcs.proto file via `protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative raft.proto`
