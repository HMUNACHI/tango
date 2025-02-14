#!/usr/bin/env python3
import grpc
import tango_pb2
import tango_pb2_grpc
import time

def run():
    channel = grpc.insecure_channel('localhost:50051')
    stub = tango_pb2_grpc.TangoServiceStub(channel)
    
    job_id = "job1"
    computation_graph = "dummy_graph"  
    data = b"dummy_data" 
    num_splits = 2 
    
    request = tango_pb2.TaskRequest(
        job_id=job_id,
        computation_graph=computation_graph,
        data=data,
        num_splits=num_splits
    )
    
    response = stub.SubmitTask(request)
    print("SubmitTask response:", response.message)
    
    while True:
        status_resp = stub.GetJobStatus(tango_pb2.JobStatusRequest(job_id=job_id))
        if status_resp.is_complete:
            print("Job completed:", status_resp.message)
            print("Final aggregated weights:", status_resp.final_weights)
            break
        print("Waiting for job to complete...")
        time.sleep(1)

if __name__ == '__main__':
    run()
