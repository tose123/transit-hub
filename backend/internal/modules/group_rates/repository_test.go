package group_rates

import "testing"

func TestMappedExistsWorkspacePredicateScopesAdminAccount(t *testing.T) {
	if mappedExistsWorkspacePredicate != "states.admin_account_id = $2" {
		t.Fatalf("mapped workspace predicate = %q", mappedExistsWorkspacePredicate)
	}
}
