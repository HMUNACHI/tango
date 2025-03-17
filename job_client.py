import argparse
import json
import logging
import random
import time
import grpc
import numpy as np

import protobuff_pb2 as pb
import protobuff_pb2_grpc as pb_grpc

def multiply_full(A, B, scale):
    return np.dot(A, B) * scale

def generate_matrix(rows, cols, factor):
    return np.random.uniform(0, 1, size=(rows, cols)).astype(np.float32) * factor

def parse_matrix(s):
    rows = []
    for line in s.strip().splitlines():
        if line:
            rows.append([np.float32(val) for val in line.split()])
    return np.array(rows, dtype=np.float32)

def init_client(tango_address):
    channel = grpc.insecure_channel(tango_address)
    client = pb_grpc.TangoServiceStub(channel)
    token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3NDA5NDcyMzgsImlzc3VlciI6IkNhY3R1cyBFZGdlIn0.bfq0u8921So9E9ra8Qdh6nGph0XaRCyMHZaDEDn3cu8"
    metadata = [('cactus-token', token)]
    return client, metadata, channel

def submit_job(client, metadata):
    jobID = f"Job{random.randint(0, 10000)}"
    M = N = D = 12
    row_splits, col_splits = 4, 4

    aMatrix = generate_matrix(M, D, 0.1)
    bMatrix = generate_matrix(D, N, 0.1)
    
    aBytes = json.dumps(aMatrix.tolist()).encode('utf-8')
    bBytes = json.dumps(bMatrix.tolist()).encode('utf-8')

    jobReq = pb.TaskRequest(
        consumer_id="234s5c2",
        job_id=jobID,
        operation="scaled_matmul",
        a_data=aBytes,
        b_data=bBytes,
        scale_scalar=1.0,
        row_splits=row_splits,
        col_splits=col_splits,
        m=M,
        n=N,
        d=D
    )
    response = client.SubmitTask(jobReq, metadata=metadata, timeout=10)
    if not response.accepted:
        raise Exception(f"Job {jobID} rejected: {response.message}")
    return jobID, aMatrix, bMatrix

def poll_job_status(client, metadata, jobID, aMatrix, bMatrix):
    timeout = 100.0
    while timeout > 0:
        statusReq = pb.JobStatusRequest(job_id=jobID)
        status = client.GetJobStatus(statusReq, metadata=metadata, timeout=10)
        if status.is_complete:
            expectedMatrix = multiply_full(aMatrix, bMatrix, 1.0)
            finalMatrix = parse_matrix(status.final_result.decode('utf-8'))
            if np.allclose(expectedMatrix, finalMatrix, atol=1e-5):
                logging.info("Verification passed: matrices are all close.")
            else:
                logging.error("Verification failed: matrices differ.")
            return
        time.sleep(0.1)
        timeout -= 0.1
    raise Exception(f"Job {jobID} did not complete within expected time.")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--tango-address", default="localhost:50051",
                        help="Address of the Tango service")
    args = parser.parse_args()
    logging.basicConfig(level=logging.INFO)
    logging.info(f"Using tango address: {args.tango_address}")

    client, metadata, channel = init_client(args.tango_address)
    try:
        jobID, aMatrix, bMatrix = submit_job(client, metadata)
        poll_job_status(client, metadata, jobID, aMatrix, bMatrix)
    finally:
        channel.close()

if __name__ == "__main__":
    main()
