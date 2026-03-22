package main

import (
	"fmt"
	"net"
	"oasis-data/adapters/sqlite" // or whichever adapter is configured
	"oasis-data/interfaces"

	"zhangbolong/OASIS-core/core"
	"zhangbolong/OASIS-hr/service"
	pb "zhangbolong/OASIS-hr/proto"

	"google.golang.org/grpc"
)

// HRModule implements the core.Module interface
type HRModule struct {
	server       *grpc.Server
	listener     net.Listener
	employeeRepo interfaces.EmployeeInterface
	deptRepo     interfaces.DepartmentInterface
}

func (m *HRModule) Name() string {
	return "OASIS-hr"
}

func (m *HRModule) Start() error {
	var err error
	m.listener, err = net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("failed to listen on :50051: %w", err)
	}

	m.server = grpc.NewServer()

	// Initialize services
	employeeService := service.NewEmployeeServiceServer(m.employeeRepo)
	deptService := service.NewDepartmentServiceServer(m.deptRepo)

	// Register with gRPC
	pb.RegisterEmployeeServiceServer(m.server, employeeService)
	pb.RegisterDepartmentServiceServer(m.server, deptService)

	fmt.Printf("Starting %s module on :50051...\n", m.Name())
	go func() {
		if err := m.server.Serve(m.listener); err != nil {
			fmt.Printf("Failed to serve gRPC server for %s: %v\n", m.Name(), err)
		}
	}()

	return nil
}

func (m *HRModule) Stop() error {
	if m.server != nil {
		m.server.GracefulStop()
	}
	fmt.Printf("%s module stopped.\n", m.Name())
	return nil
}

func main() {
	// Initialize OASIS-core Loader
	loader := core.NewModuleLoader()
	clientRegistry := core.NewInMemoryGRPCClientRegistry()

	// Initialize storage adapters (using SQLite for illustration)
	// In a real application, these might be injected or configured via environment variables
	employeeAdapter := sqlite.NewSqliteEmployeeAdapter()
	deptAdapter := sqlite.NewSqliteDepartmentAdapter()

	hrModule := &HRModule{
		employeeRepo: employeeAdapter,
		deptRepo:     deptAdapter,
	}

	// Register the module with OASIS-core
	if err := loader.Register(hrModule); err != nil {
		fmt.Printf("Failed to register module: %v\n", err)
		return
	}

	// Retrieve and start the module
	mod, err := loader.Get("OASIS-hr")
	if err != nil {
		fmt.Printf("Failed to get module: %v\n", err)
		return
	}

	if err := mod.Start(); err != nil {
		fmt.Printf("Failed to start module: %v\n", err)
		return
	}

	// Example: Register the local client with the core registry (useful for other modules)
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err == nil {
		_ = clientRegistry.RegisterClient("OASIS-hr", conn)
	}

	// Block main thread to keep running (in a real app, listen for SIGINT/SIGTERM)
	select {}
}
