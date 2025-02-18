# Language: Python
import grpc
import protobuff_pb2 as pb
import protobuff_pb2_grpc as pb_grpc
import torch
import numpy as np
import time
import random

def connect_to_tango():
    try:
        channel = grpc.insecure_channel('localhost:50051')
        return pb_grpc.TangoServiceStub(channel)
    except Exception as e:
        print("Error connecting to Tango server:", e)
        raise

def prepare_matmul_request(A, B, num_splits):
    try:
        m, d = A.shape
        d, n = B.shape
        
        # Split A into chunks along the 0th dimension.
        A_chunks = np.split(A, num_splits, axis=0)
        a_data = b"".join(chunk.tobytes() for chunk in A_chunks)

        return pb.TaskRequest(
            job_id=f"job{random.randint(0, 1000)}",
            operation="matmul",
            a_data=a_data,
            b_data=B.tobytes(),
            num_splits=num_splits,
            m=m,
            n=n,
            d=d,
        )
    except Exception as e:
        print("Error preparing matmul request:", e)
        raise

def dispatch_gather(stub, request):
    try:
        response = stub.SubmitTask(request)
        m, n = request.m, request.n

        while True:
            status_resp = stub.GetJobStatus(pb.JobStatusRequest(job_id=request.job_id))

            if status_resp.is_complete:
                return np.copy(np.frombuffer(status_resp.final_result, dtype=np.float16)).reshape((m, n))
            
            time.sleep(0.01)
    except Exception as e:
        print("Error during dispatch_gather:", e)
        raise


def torch_matmul_cactus(A, B):
    num_splits = 4
    stub = connect_to_tango()
    request = prepare_matmul_request(A.numpy(), B.numpy(), num_splits)
    result = dispatch_gather(stub, request)
    return  torch.from_numpy(result)

class CactusMatmulFunction(torch.autograd.Function):
    @staticmethod
    def forward(ctx, input, weight, bias):
        ctx.save_for_backward(input, weight)
        out = torch_matmul_cactus(input, weight)
        return out + bias

    @staticmethod
    def backward(ctx, grad_output):
        input, weight = ctx.saved_tensors
        grad_input = torch_matmul_cactus(grad_output, weight.t())
        grad_weight = torch_matmul_cactus(input.t(), grad_output)
        grad_bias = grad_output.sum(dim=0)
        return grad_input, grad_weight, grad_bias

class CactusLinearTorch(torch.nn.Module):
    def __init__(self, in_features, out_features):
        super(CactusLinearTorch, self).__init__()
        self.in_features = in_features
        self.out_features = out_features
        self.weight = torch.nn.Parameter(torch.randn(in_features, out_features, dtype=torch.float16))
        self.bias = torch.nn.Parameter(torch.randn(out_features, dtype=torch.float16))
        
    def forward(self, x):
         return CactusMatmulFunction.apply(x, self.weight, self.bias)
    

def test_torch_matmul_cactus():
    A = torch.randn(8, 8, dtype=torch.float16)
    B = torch.randn(8, 8, dtype=torch.float16)
    C = torch_matmul_cactus(A, B)
    expected_result = torch.matmul(A, B)
    assert torch.allclose(C, expected_result, atol=1e-2)
    

def run():
    A = torch.randn(8, 8, dtype=torch.float16)
    B = torch.randn(8, 8, dtype=torch.float16)

    model = CactusLinearTorch(8, 8)

    C = model(A)
    loss = C.sum()
    loss.backward()
    
    print(loss)
    
        
if __name__ == '__main__':
    run()