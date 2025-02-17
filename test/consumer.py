# Language: Python
import grpc
import protobuff_pb2 as pb
import protobuff_pb2_grpc as pb_grpc
import torch
import numpy as np
import time

def run():
    channel = grpc.insecure_channel('localhost:50051')
    stub = pb_grpc.TangoServiceStub(channel)
    
    job_id = "job1"
    operation = "matmul"
    num_splits = 4  

    # Generate random matrices using torch.
    A = torch.randn(8, 8, dtype=torch.float16)
    B = torch.randn(8, 8, dtype=torch.float16)
    
    # Split A into chunks along the 0th dimension.
    A_chunks = torch.chunk(A, num_splits, dim=0)
    a_data = b"".join(chunk.numpy().tobytes() for chunk in A_chunks)
    
    b_data = B.numpy().tobytes()
    
    request = pb.TaskRequest(
        job_id=job_id,
        operation=operation,
        a_data=a_data,
        b_data=b_data,
        num_splits=num_splits
    )
    
    response = stub.SubmitTask(request)
    print("SubmitTask response:", response.message)
    
    while True:
        status_resp = stub.GetJobStatus(pb.JobStatusRequest(job_id=job_id))
        if status_resp.is_complete:
            print("Job completed:", status_resp.message)
            final_result_np = np.frombuffer(status_resp.final_result, dtype=np.float16).reshape((8, 8))
            final_result_torch = torch.from_numpy(final_result_np)

            expected_result_torch = torch.matmul(A, B)

            if torch.allclose(final_result_torch, expected_result_torch, atol=1e-2):
                print("Verification success: Final result is accurate.")
            else:
                print("Verification failed: Final result does not match the expected result.")
            break
        time.sleep(1)
        
if __name__ == '__main__':
    run()