package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	. "github.com/seatsurfing/seatsurfing/server/api"
	. "github.com/seatsurfing/seatsurfing/server/config"
	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
	"github.com/seatsurfing/seatsurfing/server/service"
)

// A client may run more than one organization, and opening another one is
// theirs to do - the operator sells the account, not the shape of the business
// behind it. What must hold is that the organizations stay strangers to each
// other: separate people, separate areas, separate bookings. The client's
// identity is the only thing that spans them.

// createTestClient provisions a customer the way the operator does: an entry in
// the client directory, which is the operator's own organization, plus an
// administrator account in the workspace the client runs. Both carry the same
// email and the same credential, which is what makes them one identity.
func createTestClient(t *testing.T, operator *User, org *Organization, email string) *User {
	t.Helper()
	operatorOrg, err := GetOrganizationRepository().GetOne(operator.OrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	directory := CreateTestUserInOrgWithName(operatorOrg, email, UserRoleUser)
	directory.HashedPassword = "$2a$10$test-credential-of-the-client"
	if err := GetUserRepository().Update(directory); err != nil {
		t.Fatal(err)
	}
	admin := CreateTestUserInOrgWithName(org, email, UserRoleOrgAdmin)
	admin.HashedPassword = directory.HashedPassword
	if err := GetUserRepository().Update(admin); err != nil {
		t.Fatal(err)
	}
	return admin
}

func TestClientCreatesOwnOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	first := CreateTestOrg("first-workspace.com")
	admin := createTestClient(t, operator, first, "owner@first-workspace.com")
	adminLogin := LoginTestUser(admin.ID)

	payload := `{"name": "Second Workspace"}`
	req := NewHTTPRequest("POST", "/organization/my/", adminLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	secondID := res.Header().Get("X-Object-Id")
	if secondID == "" {
		t.Fatal("Expected the id of the new organization")
	}

	// It is a real organization, and the client administers it.
	second, err := GetOrganizationRepository().GetOne(secondID)
	if err != nil || second == nil {
		t.Fatalf("New organization %s cannot be read back", secondID)
	}
	if second.Name != "Second Workspace" {
		t.Fatalf("Expected 'Second Workspace', got '%s'", second.Name)
	}
	secondAdmin, err := GetUserRepository().GetByEmail(second.ID, admin.Email)
	if err != nil || secondAdmin == nil {
		t.Fatalf("The client has no account in the organization they just created")
	}
	if !service.HasPermission(secondAdmin, second.ID, PermissionOrgSettings, PermissionLevelAdmin) {
		t.Fatalf("The client is not an administrator of the organization they created")
	}
	// The credential is the one they already sign in with, so no second password.
	if secondAdmin.HashedPassword == "" || secondAdmin.HashedPassword != admin.HashedPassword {
		t.Fatal("The new account does not carry the client's own credential")
	}
}

// Having created it, the client can see it and walk into it.
func TestClientSwitchesIntoOrganizationTheyCreated(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	first := CreateTestOrg("switch-into-new.com")
	admin := createTestClient(t, operator, first, "owner@switch-into-new.com")
	adminLogin := LoginTestUser(admin.ID)

	payload := `{"name": "Branch Office"}`
	req := NewHTTPRequest("POST", "/organization/my/", adminLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	secondID := res.Header().Get("X-Object-Id")

	req = NewHTTPRequest("GET", "/user/organizations", adminLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var switcher []struct {
		OrganizationID string `json:"organizationId"`
	}
	json.Unmarshal(res.Body.Bytes(), &switcher)
	if len(switcher) != 2 {
		t.Fatalf("Expected both organizations in the switcher, got %d", len(switcher))
	}

	req = NewHTTPRequest("POST", "/user/organizations/"+secondID+"/switch", adminLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
}

// The heart of it: two organizations owned by the same client share nothing.
func TestClientOrganizationsHoldSeparateData(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	first := CreateTestOrg("separate-first.com")
	admin := createTestClient(t, operator, first, "owner@separate-first.com")
	adminLogin := LoginTestUser(admin.ID)

	// people and places that exist only in the first organization
	staff := CreateTestUserInOrg(first)
	location, _ := CreateTestLocationAndSpace(first)

	payload := `{"name": "Separate Second"}`
	req := NewHTTPRequest("POST", "/organization/my/", adminLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	secondID := res.Header().Get("X-Object-Id")

	secondAdmin, err := GetUserRepository().GetByEmail(secondID, admin.Email)
	if err != nil || secondAdmin == nil {
		t.Fatal("The client has no account in their second organization")
	}
	secondLogin := LoginTestUser(secondAdmin.ID)

	// The staff of the first organization are not people of the second.
	req = NewHTTPRequest("GET", "/user/", secondLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var users []struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	json.Unmarshal(res.Body.Bytes(), &users)
	for _, u := range users {
		if u.ID == staff.ID {
			t.Fatalf("%s belongs to the first organization but is listed in the second", u.Email)
		}
	}
	// Only the client's own administrator account lives there so far.
	if len(users) != 1 || users[0].ID != secondAdmin.ID {
		t.Fatalf("Expected the new organization to hold only its administrator, got %d users", len(users))
	}

	// Neither are its areas. A new workspace opens with a sample area of its
	// own, so what matters is not that it is empty but that nothing in it came
	// from the first organization.
	req = NewHTTPRequest("GET", "/location/", secondLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var locations []struct {
		ID string `json:"id"`
	}
	json.Unmarshal(res.Body.Bytes(), &locations)
	for _, l := range locations {
		if l.ID == location.ID {
			t.Fatal("The first organization's area is listed in the second")
		}
	}

	// And the first organization's area cannot be reached from the second.
	req = NewHTTPRequest("GET", "/location/"+location.ID, secondLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

// A client's organization listing is their own. It is not the operator's view
// of the customer base, so another client's organization never appears in it.
func TestClientOrganizationListShowsOnlyTheirOwn(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()

	mine := CreateTestOrg("mine-listing.com")
	admin := createTestClient(t, operator, mine, "owner@mine-listing.com")
	adminLogin := LoginTestUser(admin.ID)
	theirs := CreateTestOrg("theirs-listing.com")
	createTestClient(t, operator, theirs, "owner@theirs-listing.com")

	payload := `{"name": "Mine Too"}`
	req := NewHTTPRequest("POST", "/organization/my/", adminLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)
	mineTooID := res.Header().Get("X-Object-Id")

	req = NewHTTPRequest("GET", "/organization/", adminLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)
	var resBody []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 2 {
		t.Fatalf("Expected the client's two organizations, got %d", len(resBody))
	}
	seen := map[string]bool{}
	for _, o := range resBody {
		if o.ID == theirs.ID {
			t.Fatalf("Another client's organization '%s' is listed", o.Name)
		}
		if o.ID == operator.OrganizationID {
			t.Fatalf("The operator's own organization is listed to a client")
		}
		seen[o.ID] = true
	}
	if !seen[mine.ID] || !seen[mineTooID] {
		t.Fatal("The client's own organizations are missing from their listing")
	}
}

// Opening a workspace is an administrator's act. Someone who only books a desk
// cannot make one.
func TestPlainUserCannotCreateOrganization(t *testing.T) {
	ClearTestDB()
	CreateTestUserPlatformOperator()
	org := CreateTestOrg("no-org-for-you.com")
	user := CreateTestUserInOrg(org)
	userLogin := LoginTestUser(user.ID)

	payload := `{"name": "Not Allowed"}`
	req := NewHTTPRequest("POST", "/organization/my/", userLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

// A client can hand administrator rights to a colleague so the two of them run
// the workspace together. That colleague is not the platform's customer, and a
// workspace they opened would answer to nobody the operator has on file.
func TestPromotedStaffCannotCreateOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	org := CreateTestOrg("promoted-staff.com")
	createTestClient(t, operator, org, "owner@promoted-staff.com")

	colleague := CreateTestUserInOrgWithName(org, "colleague@promoted-staff.com", UserRoleOrgAdmin)
	colleagueLogin := LoginTestUser(colleague.ID)

	payload := `{"name": "Not Their Workspace"}`
	req := NewHTTPRequest("POST", "/organization/my/", colleagueLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

// Administering one organization confers nothing over another. The deletion
// request returns its confirmation code straight to the caller, so authorizing
// against the caller's own organization instead of the one named in the URL
// would hand a stranger the means to close a workspace that is not theirs.
func TestOrgAdminCannotDeleteAnotherOrganization(t *testing.T) {
	ClearTestDB()
	CreateTestUserPlatformOperator()
	allowOrgDelete := GetConfig().AllowOrgDelete
	GetConfig().AllowOrgDelete = true
	defer func() { GetConfig().AllowOrgDelete = allowOrgDelete }()

	mine := CreateTestOrg("attacker-org.com")
	admin := CreateTestUserOrgAdmin(mine)
	adminLogin := LoginTestUser(admin.ID)
	victim := CreateTestOrg("victim-org.com")
	CreateTestUserOrgAdmin(victim)

	req := NewHTTPRequest("DELETE", "/organization/"+victim.ID, adminLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	if org, err := GetOrganizationRepository().GetOne(victim.ID); err != nil || org == nil {
		t.Fatal("The victim's organization did not survive a stranger's delete request")
	}
}

// The operator does not create workspaces from inside their own organization:
// client organizations are created through the client record, and the operator
// has no workspace of their own to branch from.
func TestOperatorCannotCreateOrganizationForThemselves(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	payload := `{"name": "Operator Workspace"}`
	req := NewHTTPRequest("POST", "/organization/my/", operatorLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}
