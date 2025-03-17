import argparse
import json
import logging
import random
import time
import grpc

# Import generated protobuf modules
import protobuff_pb2 as pb
import protobuff_pb2_grpc as pb_grpc

# ...existing code...

def matrix_to_string(mat):
    return "\n".join(" ".join(f"{v:.2f}" for v in row) for row in mat)

def multiply_full(A, B, scale):
    m, d = len(A), len(B)
    n = len(B[0])
    C = [[0.0 for _ in range(n)] for _ in range(m)]
    for i in range(m):
        for j in range(n):
            s = sum(A[i][k] * B[k][j] for k in range(d))
            C[i][j] = s * scale
    return C

def generate_matrix(rows, cols, factor):
    return [[factor * random.uniform(0, 1) for _ in range(cols)] for _ in range(rows)]

def parse_matrix(s):
    result = []
    for line in s.strip().splitlines():
        if line:
            result.append([float(val) for val in line.split()])
    return result

def init_client(tango_address):
    # Create a simple insecure channel
    channel = grpc.insecure_channel(tango_address)
    client = pb_grpc.TangoServiceStub(channel)
    # Assume a test token is provided by a similar tango function;
    # here we use a dummy token.
    token = "dummy"
    metadata = [('cactus-token', token)]
    return client, metadata, channel

def submit_job(client, metadata):
    jobID = f"Job{random.randint(0, 10000)}"
    M = N = D = 256
    row_splits, col_splits = 4, 4

    aMatrix = generate_matrix(M, D, 0.1)
    bMatrix = generate_matrix(D, N, 0.1)
    aBytes = json.dumps(aMatrix).encode('utf-8')
    bBytes = json.dumps(bMatrix).encode('utf-8')

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
            # Assuming final_result bytes is a string representation of the matrix.
            finalMatrix = parse_matrix(status.final_result.decode('utf-8'))
            if len(expectedMatrix) != len(finalMatrix):
                logging.error("Verification failed: row count mismatch")
            else:
                tol = 0.005
                passed = True
                for i, exp_row in enumerate(expectedMatrix):
                    if len(exp_row) != len(finalMatrix[i]):
                        logging.error(f"Row {i} column mismatch")
                        passed = False
                        break
                    for j, exp_val in enumerate(exp_row):
                        if abs(exp_val - finalMatrix[i][j]) > tol:
                            logging.error(f"Mismatch at [{i}][{j}]: expected {exp_val:.2f}, got {finalMatrix[i][j]:.2f}")
                            passed = False
                if passed:
                    logging.info("Verification passed: final result matches expected matrix")
                else:
                    logging.error("Verification failed: matrices differ")
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
