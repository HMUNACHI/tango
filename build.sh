go get -u google.golang.org/grpc
go get -u google.golang.org/protobuf
export PATH="$PATH:$(go env GOPATH)/bin"

go mod tidy

protoc -I. \
  --go_out=grpc_proto/go --go_opt=paths=source_relative \
  --go-grpc_out=grpc_proto/go --go-grpc_opt=paths=source_relative \
  tango.proto

python3 -m pip3 install --upgrade pip3
pip3 install grpcio grpcio-tools

python3 -m grpc_tools.protoc -I. \
  --python_out=test \
  --grpc_python_out=test \
  tango.proto