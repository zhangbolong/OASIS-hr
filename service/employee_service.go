package service

import (
	"context"
	"fmt"
	"oasis-data/interfaces"
	"oasis-data/models/hr"

	pb "zhangbolong/OASIS-hr/proto"
)

// EmployeeServiceServerImpl implements the EmployeeService gRPC server.
type EmployeeServiceServerImpl struct {
	pb.UnimplementedEmployeeServiceServer
	EmployeeRepo interfaces.EmployeeInterface
}

func NewEmployeeServiceServer(employeeRepo interfaces.EmployeeInterface) *EmployeeServiceServerImpl {
	return &EmployeeServiceServerImpl{
		EmployeeRepo: employeeRepo,
	}
}

// CreateEmployee creates a new employee
func (s *EmployeeServiceServerImpl) CreateEmployee(ctx context.Context, req *pb.CreateEmployeeRequest) (*pb.Employee, error) {
	if req.Employee == nil {
		return nil, fmt.Errorf("employee is required")
	}

	model := &hr.Employee{
		ID:            req.Employee.Id,
		FirstName:     req.Employee.FirstName,
		LastName:      req.Employee.LastName,
		Email:         req.Employee.Email,
		PhoneNumber:   req.Employee.PhoneNumber,
		DepartmentID:  req.Employee.DepartmentId,
		Role:          req.Employee.Role,
		Level:         int(req.Employee.Level),
		Status:        req.Employee.Status,
		StartDate:     req.Employee.StartDate,
		EndDate:       req.Employee.EndDate,
		ManagerID:     req.Employee.ManagerId,
		DirectReports: req.Employee.DirectReports,
	}

	if err := s.EmployeeRepo.Create(model); err != nil {
		return nil, err
	}
	return req.Employee, nil
}

// GetEmployee retrieves an employee by ID
func (s *EmployeeServiceServerImpl) GetEmployee(ctx context.Context, req *pb.GetEmployeeRequest) (*pb.Employee, error) {
	emp, err := s.EmployeeRepo.GetByID(req.Id)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, fmt.Errorf("employee not found")
	}
	
	return s.convertToPB(emp), nil
}

// UpdateEmployee updates an employee's data
func (s *EmployeeServiceServerImpl) UpdateEmployee(ctx context.Context, req *pb.UpdateEmployeeRequest) (*pb.Employee, error) {
	if req.Employee == nil {
		return nil, fmt.Errorf("employee is required")
	}

	model := &hr.Employee{
		ID:            req.Id,
		FirstName:     req.Employee.FirstName,
		LastName:      req.Employee.LastName,
		Email:         req.Employee.Email,
		PhoneNumber:   req.Employee.PhoneNumber,
		DepartmentID:  req.Employee.DepartmentId,
		Role:          req.Employee.Role,
		Level:         int(req.Employee.Level),
		Status:        req.Employee.Status,
		StartDate:     req.Employee.StartDate,
		EndDate:       req.Employee.EndDate,
		ManagerID:     req.Employee.ManagerId,
		DirectReports: req.Employee.DirectReports,
	}

	if err := s.EmployeeRepo.Update(model); err != nil {
		return nil, err
	}
	return s.convertToPB(model), nil
}

// DeleteEmployee deletes an employee by ID
func (s *EmployeeServiceServerImpl) DeleteEmployee(ctx context.Context, req *pb.DeleteEmployeeRequest) (*pb.Empty, error) {
	if err := s.EmployeeRepo.Delete(req.Id); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// ListEmployees lists employees with optional active status filtering
func (s *EmployeeServiceServerImpl) ListEmployees(ctx context.Context, req *pb.ListEmployeesRequest) (*pb.ListEmployeesResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	employees, nextToken, err := s.EmployeeRepo.List(limit, req.Token)
	if err != nil {
		return nil, err
	}

	var pbEmployees []*pb.EmployeeBrief
	for _, emp := range employees {
		if req.SkipInactive && emp.Status != "Active" {
			continue
		}
		pbEmployees = append(pbEmployees, &pb.EmployeeBrief{
			Id:           emp.ID,
			FirstName:    emp.FirstName,
			LastName:     emp.LastName,
			Role:         emp.Role,
			Level:        int32(emp.Level),
			DepartmentId: emp.DepartmentID,
			ManagerId:    emp.ManagerID,
			Status:       emp.Status,
		})
	}

	return &pb.ListEmployeesResponse{
		Employees: pbEmployees,
		NextToken: nextToken,
	}, nil
}

// GetDirectManager retrieves an employee's direct manager
func (s *EmployeeServiceServerImpl) GetDirectManager(ctx context.Context, req *pb.GetDirectManagerRequest) (*pb.Employee, error) {
	emp, err := s.EmployeeRepo.GetByID(req.EmployeeId)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, fmt.Errorf("employee not found")
	}

	if emp.ManagerID == "" {
		return nil, fmt.Errorf("employee has no manager")
	}

	manager, err := s.EmployeeRepo.GetByID(emp.ManagerID)
	if err != nil {
		return nil, err
	}
	if manager == nil {
		return nil, fmt.Errorf("manager not found")
	}

	return s.convertToPB(manager), nil
}

// SetManager sets a new manager for an employee, handling cycle detection and dual-link atomic updates
func (s *EmployeeServiceServerImpl) SetManager(ctx context.Context, req *pb.SetManagerRequest) (*pb.Empty, error) {
	child, err := s.EmployeeRepo.GetByID(req.EmployeeId)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, fmt.Errorf("employee not found")
	}

	var toUpdate []*hr.Employee

	// If no longer setting a new manager just returning error
	if req.ManagerId == "" {
		return nil, fmt.Errorf("manager_id cannot be empty use appropriate remove method")
	}

	// Cycle Detection
	if err := CheckEmployeeCycle(s.EmployeeRepo, req.EmployeeId, req.ManagerId); err != nil {
		return nil, err
	}

	// Fetch old manager if exists
	if child.ManagerID != "" {
		oldManager, err := s.EmployeeRepo.GetByID(child.ManagerID)
		if err == nil && oldManager != nil {
			oldManager.DirectReports = removeStringFromSlice(oldManager.DirectReports, child.ID)
			toUpdate = append(toUpdate, oldManager)
		}
	}

	// Fetch new manager
	newManager, err := s.EmployeeRepo.GetByID(req.ManagerId)
	if err != nil {
		return nil, err
	}
	if newManager == nil {
		return nil, fmt.Errorf("new manager not found")
	}

	newManager.DirectReports = addStringToSlice(newManager.DirectReports, child.ID)
	child.ManagerID = newManager.ID

	toUpdate = append(toUpdate, child, newManager)

	if err := s.EmployeeRepo.UpdateEmployeeHierarchy(toUpdate); err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// GetDirectReports retrieves the list of direct reports IDs for a manager
func (s *EmployeeServiceServerImpl) GetDirectReports(ctx context.Context, req *pb.GetDirectReportsRequest) (*pb.GetDirectReportsResponse, error) {
	emp, err := s.EmployeeRepo.GetByID(req.EmployeeId)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, fmt.Errorf("manager not found")
	}

	return &pb.GetDirectReportsResponse{
		DirectReports: emp.DirectReports,
	}, nil
}

// AddDirectReport is effectively similar to SetManager, setting the child's manager to req.ManagerId
func (s *EmployeeServiceServerImpl) AddDirectReport(ctx context.Context, req *pb.AddDirectReportRequest) (*pb.Empty, error) {
	return s.SetManager(ctx, &pb.SetManagerRequest{
		EmployeeId: req.EmployeeId,
		ManagerId:  req.ManagerId,
	})
}

// RemoveDirectReport removes a child from a manager's direct reports and clears their manager ID
func (s *EmployeeServiceServerImpl) RemoveDirectReport(ctx context.Context, req *pb.RemoveDirectReportRequest) (*pb.Empty, error) {
	child, err := s.EmployeeRepo.GetByID(req.EmployeeId)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, fmt.Errorf("employee not found")
	}

	if child.ManagerID != req.ManagerId {
		return nil, fmt.Errorf("employee does not report to this manager")
	}

	manager, err := s.EmployeeRepo.GetByID(req.ManagerId)
	if err != nil {
		return nil, err
	}
	if manager == nil {
		return nil, fmt.Errorf("manager not found")
	}

	manager.DirectReports = removeStringFromSlice(manager.DirectReports, child.ID)
	child.ManagerID = ""

	toUpdate := []*hr.Employee{child, manager}
	if err := s.EmployeeRepo.UpdateEmployeeHierarchy(toUpdate); err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// GetEmployeeTree returns 1-level subtree of a manager
func (s *EmployeeServiceServerImpl) GetEmployeeTree(ctx context.Context, req *pb.GetEmployeeTreeRequest) (*pb.GetEmployeeTreeResponse, error) {
	emp, err := s.EmployeeRepo.GetByID(req.ManagerId)
	if err != nil {
		return nil, err
	}
	if emp == nil {
		return nil, fmt.Errorf("manager not found")
	}

	return &pb.GetEmployeeTreeResponse{
		DirectReports: emp.DirectReports,
	}, nil
}

func (s *EmployeeServiceServerImpl) convertToPB(emp *hr.Employee) *pb.Employee {
	return &pb.Employee{
		Id:            emp.ID,
		FirstName:     emp.FirstName,
		LastName:      emp.LastName,
		Email:         emp.Email,
		PhoneNumber:   emp.PhoneNumber,
		DepartmentId:  emp.DepartmentID,
		Role:          emp.Role,
		Level:         int32(emp.Level),
		Status:        emp.Status,
		StartDate:     emp.StartDate,
		EndDate:       emp.EndDate,
		ManagerId:     emp.ManagerID,
		DirectReports: emp.DirectReports,
	}
}
