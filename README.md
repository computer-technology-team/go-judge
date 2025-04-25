# Go-Judge

A scalable, secure, and high-performance online judge system for evaluating Go code submissions.

## Project Architecture

Go-Judge is built with a focus on security, scalability, and performance. The system consists of several key components working together to provide a seamless code evaluation experience.

### Core Components

#### Web Server

The web server handles HTTP requests, user authentication, and serves the frontend. It's built using:

- **Go Templates**: For server-side rendering of HTML pages
- **Pure HTML/CSS/JS**: For a lightweight, fast frontend experience
- **File Embedding**: Go's file embedding feature is used to package templates and static assets
- **Chi Router**: For HTTP routing with middleware support
- **Cookie-based JWT**: For secure user authentication and session management

#### Runner Service

The runner service is responsible for executing submitted code in a secure, isolated environment:

- **Docker Isolation**: Each submission runs in its own isolated Docker container
- **Resource Limiting**: CPU and memory limits are enforced for each submission
- **Health Checks**: Ensures runner services are available and functioning
- **gRPC Communication**: For efficient communication between services
- **DNS Load Balancing**: For distributing load across multiple runners
- **Scalability**: Can be scaled horizontally to handle increased load (currently configured with 3 instances)
- **Private Network**: API is internal and private, operating within an isolated Docker Compose network

#### Submission Broker

A custom program-level broker manages the submission workflow:

- **Job Queuing**: Manages submission queue to prevent system overload
- **Resource Management**: Limits concurrent evaluations based on available resources
- **Fault Tolerance**: Handles runner failures and retries submissions when necessary
- **Status Updates**: Provides real-time updates on submission status

#### Database

- **PostgreSQL**: For persistent storage of problems, submissions, users, etc.
- **SQLC**: For type-safe SQL queries with Go code generation

### Key Features

1. **Secure Code Execution**:

   - Fully isolated Docker containers for each submission
   - Resource limits (CPU, memory, time) to prevent abuse
   - Secure execution environment to prevent system access

2. **Scalable Architecture**:

   - Multiple runner services with health checks
   - Load balancing across runners
   - Queuing system to handle traffic spikes

3. **Fault Tolerance**:

   - Automatic retry mechanism for failed submissions
   - Health monitoring of runner services
   - Graceful degradation under heavy load

4. **Developer Experience**:
   - Clean, intuitive UI for submitting and reviewing code
   - Real-time feedback on submission status
   - Detailed error messages and test case results

### Deployment

The project uses Docker Compose for local development and deployment:

- **Judge Service**: Main web server handling HTTP requests
- **Runner Services**: Multiple instances for code execution (default: 3)
- **PostgreSQL**: Database for persistent storage
- **Support Services**:
  - Image Puller: Pre-pulls required Docker images
  - Utility Volume Creator: Prepares shared volumes for runners

### Technology Stack

- **Backend**: Go
- **Frontend**: Go Templates, HTML, CSS, JavaScript
- **Database**: PostgreSQL
- **API**: gRPC (internal and private)
- **Containerization**: Docker
- **Configuration**: YAML
- **Authentication**: Cookie-based JWT
- **Routing**: Chi router
- **SQL**: SQLC for type-safe queries
- **Networking**: Isolated Docker Compose networks

## Getting Started

To run the project locally:

```bash
# Clone the repository
git clone https://github.com/computer-technology-team/go-judge.git
cd go-judge

# Start the services
docker-compose up
```
