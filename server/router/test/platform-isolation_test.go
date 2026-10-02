package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	. "github.com/seatsurfing/seatsurfing/server/api"
	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/testutil"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

// The platform operator sells the product; the organizations they provision are
// their clients' businesses. Operator power is account-scoped - the lifecycle of
// an organization - and must stop at the boundary of what happens inside it.
// These tests pin that boundary down: every request below is made by a super
// admin against a client organization they do not belong to.

func TestPlatformOperatorCannotReadClientLocation(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-locations.com")
	location, _ := CreateTestLocationAndSpace(client)

	req := NewHTTPRequest("GET", "/location/"+location.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestPlatformOperatorCannotWriteClientLocation(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-writes.com")
	location, _ := CreateTestLocationAndSpace(client)

	payload := `{"name": "Renamed by the operator", "timezone": "Europe/Berlin"}`
	req := NewHTTPRequest("PUT", "/location/"+location.ID, operatorLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestPlatformOperatorCannotReadClientUser(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-users.com")
	staff := CreateTestUserInOrg(client)

	req := NewHTTPRequest("GET", "/user/"+staff.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

// The operator's user listing is their client directory, so it must never grow
// to include the people who work for those clients.
func TestPlatformOperatorUserListExcludesClientStaff(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-directory.com")
	staff := CreateTestUserInOrg(client)

	req := NewHTTPRequest("GET", "/user/", operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	var resBody []struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	json.Unmarshal(res.Body.Bytes(), &resBody)
	for _, u := range resBody {
		if u.ID == staff.ID {
			t.Fatalf("Client staff %s leaked into the operator's user directory", staff.Email)
		}
	}
}

// Provisioning a client's admin goes through attachClient, which copies an
// identity the operator already holds. Inventing a user inside someone else's
// organization is a different power and is not granted.
func TestPlatformOperatorCannotCreateUserInClientOrg(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-provision.com")

	payload := `{"email": "planted@client-provision.com", "firstname": "Planted", "lastname": "Account", "organizationId": "` + client.ID + `", "role": 20}`
	req := NewHTTPRequest("POST", "/user/", operatorLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	if planted, err := GetUserRepository().GetByEmail(client.ID, "planted@client-provision.com"); err == nil && planted != nil {
		t.Fatalf("Operator planted a user inside a client organization")
	}
}

func TestPlatformOperatorCannotReadClientPresenceReport(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-report.com")
	location, _ := CreateTestLocationAndSpace(client)

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	req := NewHTTPRequest("GET", "/booking/report/presence/?start="+url.QueryEscape(start.Format(JsDateTimeFormatWithTimezone))+
		"&end="+url.QueryEscape(end.Format(JsDateTimeFormatWithTimezone))+
		"&locationId="+location.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

// A client's directory entry lives in the operator's organization so that the
// operator has something to list. That is bookkeeping, and the client must
// never see it as a workspace of theirs.
func TestClientSwitcherHidesPlatformOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorOrg, err := GetOrganizationRepository().GetOne(operator.OrganizationID)
	if err != nil {
		t.Fatal(err)
	}

	// the client: a directory entry next to the operator, plus the workspace
	// they actually administer
	email := "client@switcher.com"
	CreateTestUserInOrgWithName(operatorOrg, email, UserRoleUser)
	clientOrg := CreateTestOrg("switcher-client.com")
	admin := CreateTestUserInOrgWithName(clientOrg, email, UserRoleOrgAdmin)
	adminLogin := LoginTestUser(admin.ID)

	req := NewHTTPRequest("GET", "/user/organizations", adminLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	var resBody []struct {
		OrganizationID   string `json:"organizationId"`
		OrganizationName string `json:"organizationName"`
	}
	json.Unmarshal(res.Body.Bytes(), &resBody)
	for _, o := range resBody {
		if o.OrganizationID == operatorOrg.ID {
			t.Fatalf("The operator's organization %s is offered to a client", o.OrganizationName)
		}
	}
	if len(resBody) != 1 || resBody[0].OrganizationID != clientOrg.ID {
		t.Fatalf("Expected only the client's own organization, got %d", len(resBody))
	}
}

// ... and they cannot reach it by asking for it directly either.
func TestClientCannotSwitchIntoPlatformOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorOrg, err := GetOrganizationRepository().GetOne(operator.OrganizationID)
	if err != nil {
		t.Fatal(err)
	}

	email := "client@noswitch.com"
	CreateTestUserInOrgWithName(operatorOrg, email, UserRoleUser)
	clientOrg := CreateTestOrg("noswitch-client.com")
	admin := CreateTestUserInOrgWithName(clientOrg, email, UserRoleOrgAdmin)
	adminLogin := LoginTestUser(admin.ID)

	req := NewHTTPRequest("POST", "/user/organizations/"+operatorOrg.ID+"/switch", adminLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

// The operator's list is client organizations only - their own is not a
// workspace and never appears among them.
func TestOrganizationListExcludesPlatformOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)
	client := CreateTestOrg("listed-client.com")

	req := NewHTTPRequest("GET", "/organization/", operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	var resBody []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(res.Body.Bytes(), &resBody)
	if len(resBody) != 1 || resBody[0].ID != client.ID {
		t.Fatalf("Expected only the client organization, got %d entries", len(resBody))
	}
	for _, o := range resBody {
		if o.ID == operator.OrganizationID {
			t.Fatalf("The operator's own organization is listed as a client organization")
		}
	}
}

// Deleting a client deletes the business: the workspace, its areas and
// bookings, and the staff who worked in it. Anything left behind would be a
// workspace nobody owns that its former staff could still sign in to.
func TestDeletingClientDeletesTheirOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorOrg, err := GetOrganizationRepository().GetOne(operator.OrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	operatorLogin := LoginTestUser(operator.ID)

	email := "client@cascade.com"
	directory := CreateTestUserInOrgWithName(operatorOrg, email, UserRoleUser)
	clientOrg := CreateTestOrg("cascade-client.com")
	CreateTestUserInOrgWithName(clientOrg, email, UserRoleOrgAdmin)
	staff := CreateTestUserInOrgWithName(clientOrg, "staff@cascade.com", UserRoleUser)
	location, _ := CreateTestLocationAndSpace(clientOrg)

	req := NewHTTPRequest("DELETE", "/user/"+directory.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	if org, err := GetOrganizationRepository().GetOne(clientOrg.ID); err == nil && org != nil {
		t.Fatalf("The client's organization survived their deletion")
	}
	if u, err := GetUserRepository().GetOne(staff.ID); err == nil && u != nil {
		t.Fatalf("Staff of the deleted client can still sign in")
	}
	if l, err := GetLocationRepository().GetOne(location.ID); err == nil && l != nil {
		t.Fatalf("The deleted client's areas survived")
	}
}

// ... but a workspace two clients share belongs to both of them, so removing
// one customer must not destroy the other customer's data.
func TestDeletingClientKeepsOrganizationSharedWithAnotherClient(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorOrg, err := GetOrganizationRepository().GetOne(operator.OrganizationID)
	if err != nil {
		t.Fatal(err)
	}
	operatorLogin := LoginTestUser(operator.ID)

	shared := CreateTestOrg("shared-org.com")

	leaving := "leaving@shared.com"
	leavingDirectory := CreateTestUserInOrgWithName(operatorOrg, leaving, UserRoleUser)
	leavingAdmin := CreateTestUserInOrgWithName(shared, leaving, UserRoleOrgAdmin)

	staying := "staying@shared.com"
	CreateTestUserInOrgWithName(operatorOrg, staying, UserRoleUser)
	stayingAdmin := CreateTestUserInOrgWithName(shared, staying, UserRoleOrgAdmin)

	req := NewHTTPRequest("DELETE", "/user/"+leavingDirectory.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	if org, err := GetOrganizationRepository().GetOne(shared.ID); err != nil || org == nil {
		t.Fatalf("A shared organization was destroyed by deleting one of its clients")
	}
	if u, err := GetUserRepository().GetOne(stayingAdmin.ID); err != nil || u == nil {
		t.Fatalf("The remaining client lost their access to the shared organization")
	}
	if u, err := GetUserRepository().GetOne(leavingAdmin.ID); err == nil && u != nil {
		t.Fatalf("The deleted client kept their access to the shared organization")
	}
}

// The operator removes a client's workspace outright. The emailed confirmation
// code is for an organization deleting itself, and cannot even be sent now that
// organizations have no domain to build a link from.
func TestOperatorDeletesOrganizationWithoutEmailConfirmation(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)
	client := CreateTestOrg("deleted-outright.com")
	CreateTestLocationAndSpace(client)

	req := NewHTTPRequest("DELETE", "/organization/"+client.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	if org, err := GetOrganizationRepository().GetOne(client.ID); err == nil && org != nil {
		t.Fatalf("Organization still exists after the operator deleted it")
	}
}

// The platform's own organization holds the operator's account: deleting it
// would delete the ability to run the platform.
func TestOperatorCannotDeleteOwnOrganization(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	req := NewHTTPRequest("DELETE", "/organization/"+operator.OrganizationID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)

	if org, err := GetOrganizationRepository().GetOne(operator.OrganizationID); err != nil || org == nil {
		t.Fatalf("The platform's own organization was deleted")
	}
}

// Deleting one of a client's staff must leave nothing of them behind - not a
// booking, not a group membership, and above all not a live session or refresh
// token, which would still be a way into the workspace.
func TestDeletingStaffLeavesNothingBehind(t *testing.T) {
	ClearTestDB()
	org := CreateTestOrg("staff-cleanup.com")
	admin := CreateTestUserOrgAdmin(org)
	adminLogin := LoginTestUser(admin.ID)

	staff := CreateTestUserInOrg(org)
	staffLogin := LoginTestUser(staff.ID)
	// LoginTestUser is a stub that mints no real credentials, so the traces a
	// signed-in user leaves are written directly - they are the point of this
	// test, and asserting on rows that were never created would prove nothing.
	var sessionID string
	if err := GetDatabase().DB().QueryRow("INSERT INTO sessions (user_id, device, created) "+
		"VALUES ($1, 'test device', NOW()) RETURNING id", staff.ID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetDatabase().DB().Exec("INSERT INTO refresh_tokens (user_id, created, expiry, session_id) "+
		"VALUES ($1, NOW(), NOW() + INTERVAL '30 days', $2)", staff.ID, sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetDatabase().DB().Exec("INSERT INTO auth_attempts (organization_id, user_id, email, timestamp, successful) "+
		"VALUES ($1, $2, $3, NOW(), TRUE)", org.ID, staff.ID, staff.Email); err != nil {
		t.Fatal(err)
	}
	group := CreateTestGroup(org, staff)
	location, space := CreateTestLocationAndSpace(org)
	GetSettingsRepository().Set(org.ID, SettingMaxDaysInAdvance.Name, "5000")

	// a booking of theirs
	payload := `{"spaceId": "` + space.ID + `", "enter": "2030-09-02T08:30:00Z", "leave": "2030-09-02T17:00:00Z", "subject": "Test Event"}`
	req := NewHTTPRequest("POST", "/booking/", staffLogin.UserID, bytes.NewBufferString(payload))
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusCreated, res.Code)

	req = NewHTTPRequest("DELETE", "/user/"+staff.ID, adminLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	leftovers := map[string]string{
		"bookings":          "SELECT COUNT(*) FROM bookings WHERE user_id = $1",
		"users_groups":      "SELECT COUNT(*) FROM users_groups WHERE user_id = $1",
		"users_preferences": "SELECT COUNT(*) FROM users_preferences WHERE user_id = $1",
		"refresh_tokens":    "SELECT COUNT(*) FROM refresh_tokens WHERE user_id = $1",
		"sessions":          "SELECT COUNT(*) FROM sessions WHERE user_id = $1",
		"auth_attempts":     "SELECT COUNT(*) FROM auth_attempts WHERE user_id = $1",
	}
	for table, query := range leftovers {
		var num int
		if err := GetDatabase().DB().QueryRow(query, staff.ID).Scan(&num); err != nil {
			t.Fatal(err)
		}
		if num != 0 {
			t.Fatalf("%d row(s) left in %s for a deleted user", num, table)
		}
	}

	// the workspace itself is untouched: one person leaving is not the company
	if l, err := GetLocationRepository().GetOne(location.ID); err != nil || l == nil {
		t.Fatalf("Deleting a staff member removed the organization's areas")
	}
	if g, err := GetGroupRepository().GetOne(group.ID); err != nil || g == nil {
		t.Fatalf("Deleting a staff member removed the organization's groups")
	}
}

// Deleting an organization must leave no trace of the people who worked in it.
func TestDeletingOrganizationLeavesNothingBehind(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	org := CreateTestOrg("org-cleanup.com")
	staff := CreateTestUserInOrg(org)
	CreateTestGroup(org, staff)
	CreateTestLocationAndSpace(org)

	var sessionID string
	if err := GetDatabase().DB().QueryRow("INSERT INTO sessions (user_id, device, created) "+
		"VALUES ($1, 'test device', NOW()) RETURNING id", staff.ID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetDatabase().DB().Exec("INSERT INTO refresh_tokens (user_id, created, expiry, session_id) "+
		"VALUES ($1, NOW(), NOW() + INTERVAL '30 days', $2)", staff.ID, sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetDatabase().DB().Exec("INSERT INTO auth_attempts (organization_id, user_id, email, timestamp, successful) "+
		"VALUES ($1, $2, $3, NOW(), TRUE)", org.ID, staff.ID, staff.Email); err != nil {
		t.Fatal(err)
	}

	req := NewHTTPRequest("DELETE", "/organization/"+org.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusNoContent, res.Code)

	orphans := map[string]string{
		"sessions":          "SELECT COUNT(*) FROM sessions s LEFT JOIN users u ON u.id = s.user_id WHERE u.id IS NULL",
		"refresh_tokens":    "SELECT COUNT(*) FROM refresh_tokens r LEFT JOIN users u ON u.id = r.user_id WHERE u.id IS NULL",
		"auth_attempts":     "SELECT COUNT(*) FROM auth_attempts a LEFT JOIN users u ON u.id = a.user_id WHERE a.user_id IS NOT NULL AND u.id IS NULL",
		"users_preferences": "SELECT COUNT(*) FROM users_preferences p LEFT JOIN users u ON u.id = p.user_id WHERE u.id IS NULL",
		"users_groups":      "SELECT COUNT(*) FROM users_groups g LEFT JOIN users u ON u.id = g.user_id WHERE u.id IS NULL",
		"bookings":          "SELECT COUNT(*) FROM bookings b LEFT JOIN users u ON u.id = b.user_id WHERE u.id IS NULL",
	}
	for table, query := range orphans {
		var num int
		if err := GetDatabase().DB().QueryRow(query).Scan(&num); err != nil {
			t.Fatal(err)
		}
		if num != 0 {
			t.Fatalf("%d orphaned row(s) left in %s after deleting an organization", num, table)
		}
	}
}

// The counterpart of the lockout: account-level management must keep working,
// otherwise the operator cannot run the platform at all.
func TestPlatformOperatorCanStillManageClientAccount(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserPlatformOperator()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-account.com")

	req := NewHTTPRequest("GET", "/organization/"+client.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	payload := `{"name": "Renamed Client", "firstname": "Foo", "lastname": "Bar", "email": "foo@seatsurfing.app", "language": "de"}`
	req = NewHTTPRequest("PUT", "/organization/"+client.ID, operatorLogin.UserID, bytes.NewBufferString(payload))
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	reloaded, err := GetOrganizationRepository().GetOne(client.ID)
	if err != nil || reloaded.Name != "Renamed Client" {
		t.Fatalf("Operator could not rename a client organization")
	}
}

// A client admin keeps full authority inside their own organization - the
// lockout above must not have narrowed the tenant's own rights.
func TestClientAdminKeepsFullAccessToOwnOrg(t *testing.T) {
	ClearTestDB()
	client := CreateTestOrg("client-self.com")
	admin := CreateTestUserOrgAdmin(client)
	adminLogin := LoginTestUser(admin.ID)

	location, _ := CreateTestLocationAndSpace(client)

	req := NewHTTPRequest("GET", "/location/"+location.ID, adminLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	staff := CreateTestUserInOrg(client)
	req = NewHTTPRequest("GET", "/user/"+staff.ID, adminLogin.UserID, nil)
	res = ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusOK, res.Code)

	var _ *User = staff
}
