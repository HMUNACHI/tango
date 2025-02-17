#!/usr/bin/env python3
import grpc
import protobuff_pb2 as pb
import protobuff_pb2_grpc as pb_grpc
import numpy as np

def run():
    # Connect to the gRPC server.
    channel = grpc.insecure_channel('localhost:50051')
    stub = pb_grpc.TangoServiceStub(channel)
    
    # Register this device as a compute provider.
    device_id = "device1"
    device_info = pb.DeviceInfo(
        device_id=device_id,
        internet_speed=50.0,  # in Mbps
        available_ram=4096,   # in MB
        cpu_usage=10,         # percent
        is_charging=True
    )
    reg_response = stub.RegisterDevice(device_info)
    print("RegisterDevice response:", reg_response.message)
    
    # Optionally update device status.
    device_status = pb.DeviceStatus(
        device_id=device_id,
        tasks_in_last_hour=0,
        cpu_usage=10.0,
        available_ram=4096,
        internet_speed=50.0,
        is_charging=True
    )
    status_response = stub.UpdateDeviceStatus(device_status)
    print("UpdateDeviceStatus response:", status_response.message)
    
    # Fetch a task from the server.
    device_request = pb.DeviceRequest(device_id=device_id)
    
    print("Fetching task...")
    try:
        task_assignment = stub.FetchTask(device_request)
        print("Fetched TaskAssignment:")
        print("  Job ID:", task_assignment.job_id)
        print("  Task ID:", task_assignment.task_id)
        print("  Operation:", task_assignment.operation)
        
        # For the "matmul" operation:
        # - a_data contains the concatenated binary chunks of A.
        # - b_data contains the full matrix B.
        #
        # In our simulation, the original A is 8x8 and is split into 4 chunks (each of shape 2x8).
        # Each float16 occupies 2 bytes so each chunk has 2*8*2 = 32 bytes.
        num_splits = 4  # As submitted by the consumer
        
        a_data = task_assignment.a_data
        b_data = task_assignment.b_data
        
        # Extract the shard index from the task_id (expected format: "jobID_shardIndex")
        task_id_parts = task_assignment.task_id.split('_')
        if len(task_id_parts) != 2:
            raise ValueError("Invalid task_id format")
        shard_index = int(task_id_parts[1])  # This is 1-indexed; adjust to 0-indexed.
        chunk_index = shard_index - 1
        
        # Calculate byte size for each A_chunk: each chunk is (2x8) with 16 elements at 2 bytes each.
        chunk_byte_size = 16 * 2  # 32 bytes
        
        start = chunk_index * chunk_byte_size
        end = start + chunk_byte_size
        a_chunk_bytes = a_data[start:end]
        
        # Convert the binary data to numpy arrays.
        # A_chunk will be of shape (2, 8) and B_full of shape (8, 8)
        A_chunk = np.frombuffer(a_chunk_bytes, dtype=np.float16).reshape(2, 8)
        B_full = np.frombuffer(b_data, dtype=np.float16).reshape(8, 8)
        
        # Perform matrix multiplication: each worker computes its result chunk.
        result_chunk = np.matmul(A_chunk, B_full)
        
        # Convert the result chunk to binary data.
        result_bytes = result_chunk.tobytes()
        
        # Create a TaskResult message with the computed binary result.
        task_result = pb.TaskResult(
            device_id=device_id,
            job_id=task_assignment.job_id,
            task_id=task_assignment.task_id,
            result_data=result_bytes
        )
        report_response = stub.ReportResult(task_result)
        print("ReportResult response:", report_response.message)
    except grpc.RpcError as e:
        print("RPC error occurred:", e)
    except Exception as e:
        print("Error processing task:", e)

if __name__ == '__main__':
    run()
