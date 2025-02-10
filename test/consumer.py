#!/usr/bin/env python3
import grpc
import tango_pb2
import tango_pb2_grpc as tango_pb2_grpc

def run():
    # Connect to the gRPC server.
    channel = grpc.insecure_channel('localhost:50051')
    stub = tango_pb2_grpc.TangoServiceStub(channel)
    
    # Define a new job submission (TaskRequest)
    job_id = "job1"
    computation_graph = "dummy_graph"  # For example, a JSON or StableHLO representation.
    data = b"dummy_data"               # Example binary data.
    num_splits = 2                     # Expecting 2 splits (i.e. 2 providers to execute this job).
    
    request = tango_pb2.TaskRequest(
        job_id=job_id,
        computation_graph=computation_graph,
        data=data,
        num_splits=num_splits
    )
    
    # Submit the job.
    response = stub.SubmitTask(request)
    print("SubmitTask response:", response.message)

if __name__ == '__main__':
    run()
