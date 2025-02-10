Step-by-step solution:
1. Generate the Python protobuf modules from tango.proto so that tango_pb2.py and tango_pb2_grpc.py exist.
2. Ensure producer.py runs in an environment (or directory) where those generated files are importable.

### Command to generate Python modules

Run the following command in your project root (/Users/henry/Desktop/tango):

protoc --python_out=. --grpc_python_out=. tango.proto
