package service

import (
	"fmt"
	"oasis-data/interfaces"
)

// CheckEmployeeCycle verifies that setting candidateManagerID as the manager of employeeID
// will not create a cycle. It traverses upwards from candidateManagerID to the top.
func CheckEmployeeCycle(employeeRepo interfaces.EmployeeInterface, employeeID, candidateManagerID string) error {
	if employeeID == candidateManagerID {
		return fmt.Errorf("employee cannot be their own manager: cycle detected for %s", employeeID)
	}
	
	currentManagerID := candidateManagerID
	for currentManagerID != "" {
		if currentManagerID == employeeID {
			return fmt.Errorf("cycle detected: %s is an ancestor of %s", employeeID, candidateManagerID)
		}
		
		manager, err := employeeRepo.GetByID(currentManagerID)
		if err != nil {
			// If a manager in the chain is not found, we can't fully verify, 
			// but we might just return an error or break. 
			// Assuming missing manager breaks the chain gracefully or is a data error.
			return fmt.Errorf("failed to fetch manager %s in hierarchy validation: %w", currentManagerID, err)
		}
		
		if manager == nil {
			break
		}
		
		currentManagerID = manager.ManagerID
	}
	
	return nil
}

// CheckDepartmentCycle verifies that setting candidateParentID as the parent of departmentID
// will not create a cycle. It traverses upwards from candidateParentID to the top.
func CheckDepartmentCycle(departmentRepo interfaces.DepartmentInterface, departmentID, candidateParentID string) error {
	if departmentID == candidateParentID {
		return fmt.Errorf("department cannot be its own parent: cycle detected for %s", departmentID)
	}
	
	currentParentID := candidateParentID
	for currentParentID != "" {
		if currentParentID == departmentID {
			return fmt.Errorf("cycle detected: %s is an ancestor of %s", departmentID, candidateParentID)
		}
		
		parent, err := departmentRepo.GetByID(currentParentID)
		if err != nil {
			return fmt.Errorf("failed to fetch parent %s in hierarchy validation: %w", currentParentID, err)
		}
		
		if parent == nil {
			break
		}
		
		currentParentID = parent.ParentID
	}
	
	return nil
}

// removeStringFromSlice is a helper to remove a specific string from a slice
func removeStringFromSlice(slice []string, val string) []string {
	var result []string
	for _, item := range slice {
		if item != val {
			result = append(result, item)
		}
	}
	return result
}

// addStringToSlice is a helper to append to a slice if not exists
func addStringToSlice(slice []string, val string) []string {
	for _, item := range slice {
		if item == val {
			return slice // already present
		}
	}
	return append(slice, val)
}
