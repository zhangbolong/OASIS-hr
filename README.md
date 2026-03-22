# OASIS-hr

This module implements the Human Resources (HR) domain for the OASIS application, providing dual-link hierarchical support for Employees and Departments.

## Features

- **Employees**: Complete CRUD, Token-based pagination listing, and hierarchy management (upward/downward links).
- **Departments**: Complete CRUD, Token-based pagination listing, and hierarchy management.
- **Transactional Updates**: Hierarchy modifications (e.g. `SetManager`) are grouped logically to be pushed to the DB in one atomic transaction, preventing data drift and O(1) constraints violations.
- **Cycle Detection**: Validations ensure that a manager cannot be assigned to an employee if the manager already reports to that employee (directly or indirectly).
- **gRPC Integration**: Completely gRPC-based communication.
- **Core Registration**: The module natively registers with the `OASIS-core` ModuleLoader.

## Setup & Compilation

### Requirements
You will need:
- Go 1.22+
- `protoc` (Protocol Buffers Compiler) 
- `protoc-gen-go` & `protoc-gen-go-grpc` plugins

Install the Go plugins if you haven't:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Generating gRPC Code

Run the following command from the `OASIS-hr` directory to compile the Protocol Buffers:

```bash
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/hr.proto
```

### Running the API

```bash
go run main.go
```

By default it listens on TCP `:50051` and registers itself to the `OASIS-core` memory loader.

### Usage Example

Since the service runs natively over gRPC, you can use `grpcurl` or a custom client to interact with it. Before interacting, ensure you have implemented the underlying DB logic in `OASIS-data` since the DB adapters use stubs by default.

**1. Create a Department:**
```bash
grpcurl -plaintext -d '{"department": {"id": "dept_1", "name": "Engineering"}}' localhost:50051 hr.DepartmentService/CreateDepartment
```

**2. List Departments:**
```bash
grpcurl -plaintext -d '{"limit": 10}' localhost:50051 hr.DepartmentService/ListDepartments
```

**3. Set an Employee's Manager:**
```bash
grpcurl -plaintext -d '{"employee_id": "emp_1", "manager_id": "emp_2"}' localhost:50051 hr.EmployeeService/SetManager
```

## Appendix: Installing `protoc`

If you do not have the base Protocol Buffers Compiler (`protoc`) installed on your system, you can install it using one of the following methods depending on your operating system:

- **macOS:** 
  ```bash
  brew install protobuf
  ```
- **Windows:** 
  - Using Scoop: `scoop install protobuf`
  - Using Chocolatey: `choco install protoc`
  - Or manually download the latest `win64.zip` from the [Protobuf GitHub Releases](https://github.com/protocolbuffers/protobuf/releases), extract it, and add the `bin` directory to your System PATH.
- **Linux (Debian/Ubuntu):** 
  ```bash
  sudo apt install protobuf-compiler
  ```
