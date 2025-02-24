#!/usr/bin/env python3
import grpc
import protobuff_pb2 as pb
import protobuff_pb2_grpc as pb_grpc
import numpy as np

class _ClientCallDetails(grpc.ClientCallDetails):
    def __init__(self, method, timeout, metadata, credentials, wait_for_ready, compression):
        self.method = method
        self.timeout = timeout
        self.metadata = metadata
        self.credentials = credentials
        self.wait_for_ready = wait_for_ready
        self.compression = compression

class TokenClientInterceptor(grpc.UnaryUnaryClientInterceptor):
    def __init__(self, token):
        self.token = token

    def intercept_unary_unary(self, continuation, client_call_details, request):
        metadata = []
        if client_call_details.metadata is not None:
            metadata = list(client_call_details.metadata)
        metadata.append(("cactus-token", self.token))
        new_details = _ClientCallDetails(
            client_call_details.method,
            client_call_details.timeout,
            metadata,
            client_call_details.credentials,
            client_call_details.wait_for_ready,
            client_call_details.compression
        )
        return continuation(new_details, request)

def run():
    channel = grpc.insecure_channel('localhost:50051')
    intercept_channel = grpc.intercept_channel(channel, TokenClientInterceptor("test4956"))  # Modified
    stub = pb_grpc.TangoServiceStub(intercept_channel)  # Modified
    
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
    
    device_request = pb.DeviceRequest(device_id=device_id)
    
    print("Fetching task...")

    while True:
        try:
            task_assignment = stub.FetchTask(device_request)
            print("Fetched TaskAssignment:")
            print("  Job ID:", task_assignment.job_id)
            print("  Task ID:", task_assignment.task_id)
            print("  Operation:", task_assignment.operation)
            
            a_data = task_assignment.a_data
            b_data = task_assignment.b_data
            m = task_assignment.m
            n = task_assignment.n
            d = task_assignment.d
            num_splits = task_assignment.num_splits
            m_chunk = m // num_splits
            bytes = 2
            
            task_id_parts = task_assignment.task_id.rsplit('_', 1)
            if len(task_id_parts) != 2:
                raise ValueError("Invalid task_id format")
            shard_index = int(task_id_parts[1])  # This is 1-indexed; adjust to 0-indexed.
            chunk_index = shard_index - 1
            
            chunk_byte_size = m_chunk * n * bytes
            
            start = chunk_index * chunk_byte_size
            end = start + chunk_byte_size
            a_chunk_bytes = a_data[start:end]
            
            A_chunk = np.frombuffer(a_chunk_bytes, dtype=np.float16).reshape(m_chunk, d)
            B_full = np.frombuffer(b_data, dtype=np.float16).reshape(d, n)
            
            result_chunk = np.matmul(A_chunk, B_full)
            
            result_bytes = result_chunk.tobytes()
            
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
            # break
        except Exception as e:
            print("Error processing task:", e)
            # break

if __name__ == '__main__':
    run()
