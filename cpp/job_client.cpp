#include <chrono>
#include <cstdlib>
#include <iostream>
#include <random>
#include <sstream>
#include <string>
#include <thread>
#include <vector>

#include <grpcpp/grpcpp.h>
#include "protobuff.grpc.pb.h"  

using grpc::Channel;
using grpc::ClientContext;
using grpc::Status;
using pb::TaskRequest;
using pb::TaskResponse;
using pb::JobStatusRequest;
using pb::JobStatusResponse;
using pb::TangoService;

const std::string kCactusToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE3NDA5NDcyMzgsImlzc3VlciI6IkNhY3R1cyBFZGdlIn0.bfq0u8921So9E9ra8Qdh6nGph0XaRCyMHZaDEDn3cu8";

std::string generateRandomJobId() {
    std::random_device rd;
    std::mt19937 gen(rd());
    int taskID = std::uniform_int_distribution<>(0, 10000)(gen);
    return "Job" + std::to_string(taskID);
}

std::vector<std::vector<float>> generateMatrix(int rows, int cols, float factor) {
    std::vector<std::vector<float>> m(rows, std::vector<float>(cols));
    std::random_device rd;
    std::mt19937 gen(rd());
    std::uniform_real_distribution<float> dist(0.0, 1.0);
    for (int i = 0; i < rows; i++) {
        for (int j = 0; j < cols; j++) {
            m[i][j] = factor * dist(gen);
        }
    }
    return m;
}

std::string matrixToJson(const std::vector<std::vector<float>>& mat) {
    std::ostringstream oss;
    oss << "[";
    for (size_t i = 0; i < mat.size(); i++) {
        oss << "[";
        for (size_t j = 0; j < mat[i].size(); j++) {
            oss << mat[i][j];
            if (j + 1 < mat[i].size()) oss << ",";
        }
        oss << "]";
        if (i + 1 < mat.size()) oss << ",";
    }
    oss << "]";
    return oss.str();
}

int main(int argc, char** argv) {
    std::string tangoAddress = "localhost:50051";
    if (argc > 1) tangoAddress = argv[1];

    // Create an insecure channel.
    auto channel = grpc::CreateChannel(tangoAddress, grpc::InsecureChannelCredentials());
    std::unique_ptr<TangoService::Stub> stub = TangoService::NewStub(channel);

    // Generate a job ID and matrices.
    std::string jobId = generateRandomJobId();
    int M = 256, N = 256, D = 256;
    int RowSplit = 4, ColSplit = 4;
    auto aMatrix = generateMatrix(M, D, 0.1f);
    auto bMatrix = generateMatrix(D, N, 0.1f);
    std::string aJson = matrixToJson(aMatrix);
    std::string bJson = matrixToJson(bMatrix);

    // Build task request.
    TaskRequest taskReq;
    taskReq.set_consumerid("234s5c2");
    taskReq.set_jobid(jobId);
    taskReq.set_operation("scaled_matmul");
    taskReq.set_adata(aJson);
    taskReq.set_bdata(bJson);
    taskReq.set_rowsplits(RowSplit);
    taskReq.set_colsplits(ColSplit);
    taskReq.set_m(M);
    taskReq.set_n(N);
    taskReq.set_d(D);
    taskReq.set_scalescalar(1.0f);

    // Set up client context with metadata.
    ClientContext context;
    context.AddMetadata("cactus-token", kCactusToken);
    // Set deadline for RPC.
    auto deadline = std::chrono::system_clock::now() + std::chrono::seconds(10);
    context.set_deadline(deadline);

    // Submit task.
    TaskResponse taskRes;
    Status status = stub->SubmitTask(&context, taskReq, &taskRes);
    if (!status.ok() || !taskRes.accepted()) {
        std::cerr << "SubmitTask failed: " << status.error_message() << std::endl;
        return EXIT_FAILURE;
    }
    std::cout << "Job submitted with JobID: " << jobId << std::endl;

    // Poll for job status.
    while (true) {
        ClientContext pollContext;
        pollContext.AddMetadata("cactus-token", kCactusToken);
        auto pollDeadline = std::chrono::system_clock::now() + std::chrono::seconds(10);
        pollContext.set_deadline(pollDeadline);

        JobStatusRequest statusReq;
        statusReq.set_jobid(jobId);
        JobStatusResponse statusRes;
        Status pollStatus = stub->GetJobStatus(&pollContext, statusReq, &statusRes);
        if (!pollStatus.ok()) {
            std::cerr << "GetJobStatus failed: " << pollStatus.error_message() << std::endl;
            return EXIT_FAILURE;
        }
        if (statusRes.iscomplete()) {
            std::cout << "Job complete. Final result:\n" << statusRes.finalresult() << std::endl;
            break;
        }
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
    return EXIT_SUCCESS;
}
