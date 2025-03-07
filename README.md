# Cactus Tango


## Table of Contents

- [Overview](#overview)
- [Core Purpose and Objectives](#core-purpose-and-objectives)
- [System Architecture and Workflow](#system-architecture-and-workflow)
- [Technical and Logical Flow](#technical-and-logical-flow)
- [Design Decisions and Alternatives](#design-decisions-and-alternatives)
- [Strengths, Limitations, and Risks](#strengths-limitations-and-risks)
- [Business Value and Strategic Importance](#business-value-and-strategic-importance)
- [Implementation Challenges and Mitigation Strategies](#implementation-challenges-and-mitigation-strategies)
- [Key Performance Indicators (KPIs)](#key-performance-indicators-kpis)
- [Long-Term Vision and Future Expansions](#long-term-vision-and-future-expansions)
- [Project Setup and Deployment](#project-setup-and-deployment)
  - [Install Packages](#install-packages)
  - [Build Protobuf](#build-protobuf-only-when-changes-are-made-to-tangoprototype)
  - [Running the Server Locally](#running-the-server-locally)
  - [Deploying to GCP Compute Engine](#deploying-to-gcp-compute-engine)
  - [Deploying on a Local Server](#deploying-on-a-local-server)
- [Contributing](#contributing)
- [License and Proprietary Notice](#license-and-proprietary-notice)

## Overview

Tango is a distributed computation platform designed to execute heavy matrix operations—such as scaled matrix multiplication—by partitioning tasks across multiple devices. It leverages a microservices architecture built in Go, with secure gRPC communication (using TLS) and stateless JWT authentication. The platform is optimized for scalability, operational efficiency, and seamless integration with cloud services (notably Google Cloud Platform).

## Core Purpose and Objectives

- **Distributed Computation:** Efficiently partition and distribute computationally intensive tasks (e.g., matrix multiplications) across multiple devices.
- **Scalability:** Enable dynamic scaling by leveraging parallel processing across distributed compute nodes.
- **Security:** Secure communications via TLS and JWT authentication to ensure only authorized consumers and devices participate.
- **Operational Efficiency:** Simplify operations by integrating with GCP services for secret management and logging.
- **Modular Architecture:** Establish a flexible foundation to support future distributed compute tasks beyond matrix multiplication.

## System Architecture and Workflow

### High-Level Workflow

1. **Job Submission:**  
   - Consumers submit a job through a gRPC `SubmitTask` RPC.
   - The job includes matrix data, the operation type, and parameters for task splitting.

2. **Task Distribution:**  
   - Devices poll for available tasks using the `FetchTask` RPC.
   - The server reserves a task shard for the device by splitting the matrices into smaller parts.

3. **Task Execution and Reporting:**  
   - Devices perform computation on the assigned matrix shard.
   - Results are reported back using the `ReportResult` RPC, updating the overall job status.

4. **Result Aggregation:**  
   - Once all shards have been processed, the server reassembles the complete result matrix.
   - Final transaction logs are uploaded to GCP Cloud Storage.

5. **Security Enforcement:**  
   - gRPC calls are secured with TLS.
   - JWT tokens are used to authenticate requests via a custom interceptor.


## Technical and Logical Flow

- **Initialization:**  
  The server loads configuration, establishes TLS connections using GCP-provided secrets, and initiates a background process to reap expired tasks.

- **Job Lifecycle:**  
  - **Submission:** Jobs are created from consumer requests and queued.
  - **Task Assignment:** Devices retrieve tasks; the server splits matrix data based on row/column configuration.
  - **Processing:** Devices compute their assigned shard.
  - **Aggregation:** Shard results are collected and merged into a final matrix.
  - **Finalization:** Logs are stored and the job’s final result is made available.

- **Security Flow:**  
  JWT tokens, included in request metadata, are validated against secrets stored in GCP Secret Manager before processing any RPC.

## Design Decisions and Alternatives

- **gRPC with TLS:**  
  - *Decision:* Provides low-latency, bi-directional communication and secure data transfer.
  - *Alternative Rejected:* REST/HTTP APIs were considered but deemed less efficient for real-time distributed operations.

- **JWT-Based Authentication:**  
  - *Decision:* Enables stateless and scalable authentication.
  - *Alternative Rejected:* External OAuth providers were rejected due to integration complexities with internal secret management.

- **Distributed Task Partitioning:**  
  - *Decision:* Matrix sharding allows parallel processing and efficient resource usage.
  - *Alternative Rejected:* Centralized processing was not viable for the required performance and scalability.

- **GCP Integration:**  
  - *Decision:* Use GCP Secret Manager and Cloud Storage for simplified secret and log management.
  - *Alternative Rejected:* Self-hosted solutions would add operational overhead and complexity.

- **Implementation in Go:**  
  - *Decision:* Go’s concurrency model and static binary deployment are ideal for microservices.
  - *Alternative Rejected:* Other languages were less efficient or more complex for the targeted compute environment.

## Strengths, Limitations, and Risks

### Strengths
- **Scalability:** Distributes heavy computation across multiple devices for faster processing.
- **Security:** TLS and JWT ensure secure communication and authentication.
- **Cloud-Native Design:** Simplifies operations by integrating with cloud services (GCP).

### Limitations
- **Complex Task Reassembly:** The algorithm for merging matrix shards can be intricate and may require further refinement.
- **Dependency on GCP:** Tight coupling with GCP services may impact portability.
- **Basic Logging:** CSV-based logging is simple but might not scale as well as a dedicated database system.

### Risks
- **Security Breaches:** Compromise of JWT secrets could expose the system.
- **Performance Bottlenecks:** Reassembly of tasks and job coordination may become bottlenecks under heavy load.
- **Device and Network Failures:** Distributed processing is inherently vulnerable to intermittent device or network failures.


## Business Value and Strategic Importance

- **Competitive Advantage:**  
  Accelerates compute-intensive tasks such as matrix operations, offering a significant edge in machine learning, analytics, and scientific computing.

- **Cost Efficiency:**  
  Utilizes distributed devices to optimize resource use and reduce overall compute costs.

- **Strategic Platform:**  
  Establishes a foundation for a broader distributed computing ecosystem, paving the way for additional advanced compute services.

- **Security and Trust:**  
  Strong security measures build trust with enterprise customers and safeguard sensitive operations.


## Implementation Challenges and Mitigation Strategies

- **Task Scheduling and Reassembly:**  
  *Challenge:* Ensuring correct assignment and aggregation of matrix shards.  
  *Mitigation:* Use robust reaper logic, detailed logging, and implement redundancy where necessary.

- **Security Management:**  
  *Challenge:* Securely managing secrets and JWT tokens across distributed nodes.  
  *Mitigation:* Regular secret rotation, strict access policies, and continuous monitoring of authentication events.

- **Scalability Under Load:**  
  *Challenge:* Maintaining performance as the number of concurrent tasks increases.  
  *Mitigation:* Horizontal scaling, performance stress testing, and optimization of serialization processes.

- **Monitoring and Debugging:**  
  *Challenge:* Tracking distributed task performance and diagnosing failures.  
  *Mitigation:* Integrate centralized logging and monitoring tools (e.g., Prometheus, Stackdriver) with alerting systems.


## Key Performance Indicators (KPIs)

- **Task Throughput:** Number of tasks processed per unit time.
- **End-to-End Latency:** Time from job submission to final result aggregation.
- **Success Rate:** Percentage of tasks completed without errors.
- **Resource Utilization:** Metrics on CPU, memory, and network usage.
- **Security Metrics:** Frequency of authentication failures or unauthorized access attempts.
- **Error and Reassignment Rates:** Frequency of task timeouts or failures that require reassignment.


## Long-Term Vision and Future Expansions

- **Broader Compute Tasks:**  
  Expand beyond matrix multiplication to support other operations like convolutions or data transformations.

- **Dynamic Scaling:**  
  Integrate with container orchestration platforms (e.g., Kubernetes) for auto-scaling based on workload.

- **Enhanced Fault Tolerance:**  
  Develop advanced retry and self-healing mechanisms to improve reliability.

- **Advanced Analytics:**  
  Implement machine learning for predictive task scheduling, anomaly detection, and performance optimization.

- **Multi-Cloud Support:**  
  Explore integration with other cloud providers to enhance flexibility and reduce vendor lock-in.

- **User Interface Development:**  
  Build a dashboard for real-time monitoring, job management, and performance analytics.

---

## Project Setup and Deployment

### Install Packages

1. **Install Go 1.18+:**  
   Download and install from the [official Go website](https://golang.org/dl/).

### Build Protobuf (Only When Changes Are Made to `tango.proto`)

1. **Make the Build Script Executable:**  
   ```bash
   chmod +x build.sh
