#!/bin/bash

# Directory containing the .proto files
PROTO_DIR="./protos"

# Output directory for the generated Go files
OUT_DIR="./cmd"

# Create the output directory if it doesn't exist
mkdir -p $OUT_DIR

# Generate Go bindings for each .proto file in the PROTO_DIR
for proto_file in $PROTO_DIR/*.proto; do
  protoc --go_out=$OUT_DIR --go_opt=paths=source_relative \
         --go-grpc_out=$OUT_DIR --go-grpc_opt=paths=source_relative \
         $proto_file
done

echo "Go bindings generated successfully in $OUT_DIR"