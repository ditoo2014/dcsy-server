package controller

import "testing"

func TestCirculationLogCorpScopeRequiresDepartment(t *testing.T) {
	for _, corpIDs := range [][]int{nil, {}, {0}, {-1, 0}} {
		if scope, ok := circulationLogCorpScope(corpIDs); ok || scope != nil {
			t.Fatalf("department-less user should be denied: input=%v scope=%v ok=%v", corpIDs, scope, ok)
		}
	}
}

func TestCirculationLogCorpScopeIncludesUnassignedEquipment(t *testing.T) {
	scope, ok := circulationLogCorpScope([]int{8, 8, 12})
	if !ok {
		t.Fatal("user with departments should have a query scope")
	}
	if len(scope) != 3 || scope[0] != 8 || scope[1] != 12 || scope[2] != 0 {
		t.Fatalf("unexpected department scope: %v", scope)
	}
}
