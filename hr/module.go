package hr

import (
	"fmt"
	"net"

	"oasis-data/interfaces"
	pb "zhangbolong/OASIS-hr/proto"
	"zhangbolong/OASIS-hr/service"

	"google.golang.org/grpc"
)

// HRModule implements the core.Module interface
type HRModule struct {
	server       *grpc.Server
	listener     net.Listener
	employeeRepo interfaces.EmployeeInterface
	deptRepo     interfaces.DepartmentInterface
}

func NewHRModule(employeeRepo interfaces.EmployeeInterface, deptRepo interfaces.DepartmentInterface) *HRModule {
	return &HRModule{
		employeeRepo: employeeRepo,
		deptRepo:     deptRepo,
	}
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
