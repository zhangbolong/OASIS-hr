package service

import (
	"context"
	"fmt"
	"oasis-data/interfaces"
	"oasis-data/models/hr"

	pb "zhangbolong/OASIS-hr/proto"
)

// DepartmentServiceServerImpl implements the DepartmentService gRPC server.
type DepartmentServiceServerImpl struct {
	pb.UnimplementedDepartmentServiceServer
	DepartmentRepo interfaces.DepartmentInterface
}

func NewDepartmentServiceServer(departmentRepo interfaces.DepartmentInterface) *DepartmentServiceServerImpl {
	return &DepartmentServiceServerImpl{
		DepartmentRepo: departmentRepo,
	}
}

// CreateDepartment creates a new department
func (s *DepartmentServiceServerImpl) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.Department, error) {
	if req.Department == nil {
		return nil, fmt.Errorf("department is required")
	}

	model := &hr.Department{
		ID:          req.Department.Id,
		Name:        req.Department.Name,
		ParentID:    req.Department.ParentId,
		ChildIDs:    req.Department.ChildIds,
		Description: req.Department.Description,
	}

	if err := s.DepartmentRepo.Create(model); err != nil {
		return nil, err
	}
	return req.Department, nil
}

// GetDepartment retrieves a department by ID
func (s *DepartmentServiceServerImpl) GetDepartment(ctx context.Context, req *pb.GetDepartmentRequest) (*pb.Department, error) {
	dept, err := s.DepartmentRepo.GetByID(req.Id)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, fmt.Errorf("department not found")
	}
	return s.convertToPB(dept), nil
}

// UpdateDepartment updates a department's data
func (s *DepartmentServiceServerImpl) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.Department, error) {
	if req.Department == nil {
		return nil, fmt.Errorf("department is required")
	}

	model := &hr.Department{
		ID:          req.Id,
		Name:        req.Department.Name,
		ParentID:    req.Department.ParentId,
		ChildIDs:    req.Department.ChildIds,
		Description: req.Department.Description,
	}

	if err := s.DepartmentRepo.Update(model); err != nil {
		return nil, err
	}
	return s.convertToPB(model), nil
}

// DeleteDepartment deletes a department by ID
func (s *DepartmentServiceServerImpl) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.Empty, error) {
	if err := s.DepartmentRepo.Delete(req.Id); err != nil {
		return nil, err
	}
	return &pb.Empty{}, nil
}

// ListDepartments lists departments
func (s *DepartmentServiceServerImpl) ListDepartments(ctx context.Context, req *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	departments, nextToken, err := s.DepartmentRepo.List(limit, req.Token)
	if err != nil {
		return nil, err
	}

	var pbDepartments []*pb.DepartmentBrief
	for _, dept := range departments {
		pbDepartments = append(pbDepartments, &pb.DepartmentBrief{
			Id:   dept.ID,
			Name: dept.Name,
		})
	}

	return &pb.ListDepartmentsResponse{
		Departments: pbDepartments,
		NextToken:   nextToken,
	}, nil
}

// GetParentDepartment retrieves a department's parent
func (s *DepartmentServiceServerImpl) GetParentDepartment(ctx context.Context, req *pb.GetParentDepartmentRequest) (*pb.Department, error) {
	dept, err := s.DepartmentRepo.GetByID(req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, fmt.Errorf("department not found")
	}

	if dept.ParentID == "" {
		return nil, fmt.Errorf("department has no parent")
	}

	parent, err := s.DepartmentRepo.GetByID(dept.ParentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, fmt.Errorf("parent department not found")
	}

	return s.convertToPB(parent), nil
}

// SetParentDepartment sets a new parent for a department, handling cycle detection and dual-link atomic updates
func (s *DepartmentServiceServerImpl) SetParentDepartment(ctx context.Context, req *pb.SetParentDepartmentRequest) (*pb.Empty, error) {
	child, err := s.DepartmentRepo.GetByID(req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, fmt.Errorf("department not found")
	}

	if req.ParentId == "" {
		return nil, fmt.Errorf("parent_id cannot be empty use appropriate remove method")
	}

	if err := CheckDepartmentCycle(s.DepartmentRepo, req.DepartmentId, req.ParentId); err != nil {
		return nil, err
	}

	var toUpdate []*hr.Department

	if child.ParentID != "" {
		oldParent, err := s.DepartmentRepo.GetByID(child.ParentID)
		if err == nil && oldParent != nil {
			oldParent.ChildIDs = removeStringFromSlice(oldParent.ChildIDs, child.ID)
			toUpdate = append(toUpdate, oldParent)
		}
	}

	newParent, err := s.DepartmentRepo.GetByID(req.ParentId)
	if err != nil {
		return nil, err
	}
	if newParent == nil {
		return nil, fmt.Errorf("new parent not found")
	}

	newParent.ChildIDs = addStringToSlice(newParent.ChildIDs, child.ID)
	child.ParentID = newParent.ID

	toUpdate = append(toUpdate, child, newParent)

	if err := s.DepartmentRepo.UpdateDepartmentHierarchy(toUpdate); err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// GetChildDepartments retrieves the list of child IDs for a parent department
func (s *DepartmentServiceServerImpl) GetChildDepartments(ctx context.Context, req *pb.GetChildDepartmentsRequest) (*pb.GetChildDepartmentsResponse, error) {
	dept, err := s.DepartmentRepo.GetByID(req.ParentId)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, fmt.Errorf("department not found")
	}

	return &pb.GetChildDepartmentsResponse{
		ChildIds: dept.ChildIDs,
	}, nil
}

// AddChildDepartment adds a child to a parent department
func (s *DepartmentServiceServerImpl) AddChildDepartment(ctx context.Context, req *pb.AddChildDepartmentRequest) (*pb.Empty, error) {
	return s.SetParentDepartment(ctx, &pb.SetParentDepartmentRequest{
		DepartmentId: req.DepartmentId,
		ParentId:     req.ParentId,
	})
}

// RemoveChildDepartment removes a child from a parent department
func (s *DepartmentServiceServerImpl) RemoveChildDepartment(ctx context.Context, req *pb.RemoveChildDepartmentRequest) (*pb.Empty, error) {
	child, err := s.DepartmentRepo.GetByID(req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if child == nil {
		return nil, fmt.Errorf("department not found")
	}

	if child.ParentID != req.ParentId {
		return nil, fmt.Errorf("department does not belong to this parent")
	}

	parent, err := s.DepartmentRepo.GetByID(req.ParentId)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, fmt.Errorf("parent not found")
	}

	parent.ChildIDs = removeStringFromSlice(parent.ChildIDs, child.ID)
	child.ParentID = ""

	toUpdate := []*hr.Department{child, parent}
	if err := s.DepartmentRepo.UpdateDepartmentHierarchy(toUpdate); err != nil {
		return nil, err
	}

	return &pb.Empty{}, nil
}

// GetDepartmentTree returns 1-level subtree of a parent
func (s *DepartmentServiceServerImpl) GetDepartmentTree(ctx context.Context, req *pb.GetDepartmentTreeRequest) (*pb.GetDepartmentTreeResponse, error) {
	dept, err := s.DepartmentRepo.GetByID(req.ParentId)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, fmt.Errorf("department not found")
	}

	return &pb.GetDepartmentTreeResponse{
		ChildIds: dept.ChildIDs,
	}, nil
}

func (s *DepartmentServiceServerImpl) convertToPB(dept *hr.Department) *pb.Department {
	return &pb.Department{
		Id:          dept.ID,
		Name:        dept.Name,
		ParentId:    dept.ParentID,
		ChildIds:    dept.ChildIDs,
		Description: dept.Description,
	}
}
