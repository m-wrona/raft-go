.PHONY: protos

install:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

generate:
	@rm -rf generated 2>/dev/null
	@mkdir generated
	@protoc \
		--go_out=generated \
		--go-grpc_out=require_unimplemented_servers=false:generated \
		protos/api.proto
	@protoc \
    		--go_out=generated \
    		--go-grpc_out=require_unimplemented_servers=false:generated \
    		protos/raft.proto
