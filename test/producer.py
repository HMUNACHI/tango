#!/usr/bin/env python3
import grpc
import tango_pb2
import tango_pb2_grpc as tango_pb2_grpc

def run():
    # Connect to the gRPC server.
    channel = grpc.insecure_channel('localhost:50051')
    stub = tango_pb2_grpc.TangoServiceStub(channel)
    
    # Register this device as a compute provider.
    device_id = "device1"
    device_info = tango_pb2.DeviceInfo(
        device_id=device_id,
        internet_speed=50.0,  # in Mbps
        available_ram=4096,   # in MB
        cpu_usage=10,         # percent
        is_charging=True
    )
    reg_response = stub.RegisterDevice(device_info)
    print("RegisterDevice response:", reg_response.message)
    
    # Optionally update device status.
    device_status = tango_pb2.DeviceStatus(
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
    device_request = tango_pb2.DeviceRequest(device_id=device_id)
    
    print("Fetching task...")
    try:
        task_assignment = stub.FetchTask(device_request)
        print("Fetched TaskAssignment:")
        print("  Job ID:", task_assignment.job_id)
        print("  Task ID:", task_assignment.task_id)
        print("  Computation Graph:", task_assignment.computation_graph)
        print("  Data:", task_assignment.data)
        
        # Simulate processing the task.
        # For demonstration purposes, we use a dummy weight update "1.0,2.0,3.0".
        result_data = "1.0,2.0,3.0"
        
        task_result = tango_pb2.TaskResult(
            device_id=device_id,
            job_id=task_assignment.job_id,
            task_id=task_assignment.task_id,
            result_data=result_data
        )
        report_response = stub.ReportResult(task_result)
        print("ReportResult response:", report_response.message)
    except grpc.RpcError as e:
        print("RPC error occurred:", e)

if __name__ == '__main__':
    run()
