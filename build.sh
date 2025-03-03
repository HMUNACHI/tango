if [ "$1" = "--prod" ]; then
  export ENV="production"
fi

if [ "$ENV" != "production" ]; then
  if [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    defaultCredFile="cactus-gcp-credentials.json"
    if [ -f "$defaultCredFile" ]; then
      export GOOGLE_APPLICATION_CREDENTIALS="$defaultCredFile"
    else
      echo "ERROR: GOOGLE_APPLICATION_CREDENTIALS is not set and default file ($defaultCredFile) not found."
      exit 1
    fi
  fi
fi

export PATH="$PATH:$(go env GOPATH)/bin"

go mod tidy

protoc -I. \
  --go_out=src/protobuff --go_opt=paths=source_relative \
  --go-grpc_out=src/protobuff --go-grpc_opt=paths=source_relative \
  protobuff.proto

# python3 -m pip install grpcio grpcio-tools numpy torch

# python3 -m grpc_tools.protoc -I. \
#   --python_out=test \
#   --grpc_python_out=test \
#   protobuff.proto