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
	operator := CreateTestUserSuperAdmin()
	operatorLogin := LoginTestUser(operator.ID)

	client := CreateTestOrg("client-locations.com")
	location, _ := CreateTestLocationAndSpace(client)

	req := NewHTTPRequest("GET", "/location/"+location.ID, operatorLogin.UserID, nil)
	res := ExecuteTestRequest(req)
	CheckTestResponseCode(t, http.StatusForbidden, res.Code)
}

func TestPlatformOperatorCannotWriteClientLocation(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserSuperAdmin()
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
	operator := CreateTestUserSuperAdmin()
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
	operator := CreateTestUserSuperAdmin()
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
	operator := CreateTestUserSuperAdmin()
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
	operator := CreateTestUserSuperAdmin()
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

// The counterpart of the lockout: account-level management must keep working,
// otherwise the operator cannot run the platform at all.
func TestPlatformOperatorCanStillManageClientAccount(t *testing.T) {
	ClearTestDB()
	operator := CreateTestUserSuperAdmin()
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
