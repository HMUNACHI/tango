# Tango Project Setup

## Install Packages
1. Install Go 1.18+ from the official Go website.
2. Install python 3.11 from the official website.

## Build Protobuf (only when changes are made to tango.proto)
1. Make the build scripts executable ```chmod +x build.sh```
2. Rebuild the proto buffers ```./build.sh```

## Running the Server
1. From within the tango folder, run:
   ```
   go run main.go
   ```

## Python Consumer and Producer
1. Run consumer:
   ```
   python3 test/consumer.py
   ```
2. Run producer:
   ```
   python3 test/producer.py
   ```
