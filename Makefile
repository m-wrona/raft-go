.PHONY: protos

install:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

generate:
	@rm -rf model 2>/dev/null
	@mkdir model
	@protoc \
		--go_out=model \
		--go-grpc_out=require_unimplemented_servers=false:model \
		protos/api.proto
	@protoc \
    		--go_out=model \
    		--go-grpc_out=require_unimplemented_servers=false:model \
    		protos/raft.proto
