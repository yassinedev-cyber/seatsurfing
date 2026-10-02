package router

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	. "github.com/seatsurfing/seatsurfing/server/api"
	. "github.com/seatsurfing/seatsurfing/server/config"
	. "github.com/seatsurfing/seatsurfing/server/repository"
	. "github.com/seatsurfing/seatsurfing/server/util"
)

type OrganizationRouter struct {
}

type CreateOrganizationRequest struct {
	Name      string `json:"name" validate:"required,min=2,max=64"`
	Firstname string `json:"firstname" validate:"required,min=2,max=64"`
	Lastname  string `json:"lastname" validate:"required,min=2,max=64"`
	Email     string `json:"email" validate:"required,email,max=128"`
	Language  string `json:"language" validate:"required,len=2"`
}

// CreateMyOrganizationRequest is what a client sends to open another workspace
// of their own. It carries a name and nothing else: the contact details are the
// client's own identity, which the server already knows and will not take on
// trust from the browser.
type CreateMyOrganizationRequest struct {
	Name string `json:"name" validate:"required,min=2,max=64"`
}

type GetOrganizationResponse struct {
	ID string `json:"id"`
	CreateOrganizationRequest
}

// GetOrganizationListResponse carries the numbers the platform operator needs
// to judge a client's organization at a glance. Only the listing includes them,
// so the per-organization counts are never computed on the hot path of a
// regular organization lookup.
type GetOrganizationListResponse struct {
	GetOrganizationResponse
	UserCount    int `json:"userCount"`
	BookingCount int `json:"bookingCount"`
}

type GetDomainResponse struct {
	DomainName  string     `json:"domain"`
	Active      bool       `json:"active"`
	VerifyToken string     `json:"verifyToken"`
	Primary     bool       `json:"primary"`
	Accessible  bool       `json:"accessible"`
	AccessCheck *time.Time `json:"accessCheck"`
}

type ChangeOrgEmailPayload struct {
	OrgID string `json:"orgId" validate:"required,uuid"`
	Email string `json:"email" validate:"required,email,max=256"`
	Code  int    `json:"code" validate:"required,numeric,len=6"`
}

type ChangeOrgEmailResponse struct {
	VerifyUUID string `json:"verifyUuid"`
}

type ChangeEmailAddressVerifyRequest struct {
	Code string `json:"code" validate:"required,numeric,len=6"`
}

type CompleteOrgDeletionRequest struct {
	Code string `json:"code" validate:"required,numeric,len=6"`
}

type DeleteOrgResponse struct {
	Code string `json:"code"`
}

type AuthStateOrgDeletionRequestPayload struct {
	OrganizationID string `json:"organizationId" validate:"required,uuid"`
	Code           string `json:"code" validate:"required,numeric,len=6"`
}

func (router *OrganizationRouter) SetupRoutes(s *mux.Router) {
	s.HandleFunc("/my/", router.createMyOrganization).Methods("POST")
	s.HandleFunc("/domain/verify/{domain}", router.getDomainAccessibilityToken).Methods("GET")
	s.HandleFunc("/domain/{domain}", router.getOrgForDomain).Methods("GET")
	s.HandleFunc("/{id}/domain/", router.getDomains).Methods("GET")
	s.HandleFunc("/{id}/domain/{domain}/verify", router.verifyDomain).Methods("POST")
	s.HandleFunc("/{id}/domain/{domain}/primary", router.setPrimaryDomain).Methods("POST")
	s.HandleFunc("/{id}/domain/{domain}", router.removeDomain).Methods("DELETE")
	s.HandleFunc("/{id}/domain/{domain}", router.addDomain).Methods("POST")
	s.HandleFunc("/{id}/verifyemail/{uuid}", router.verifyEmail).Methods("POST")
	s.HandleFunc("/{id}", router.getOne).Methods("GET")
	s.HandleFunc("/{id}", router.update).Methods("PUT")
	s.HandleFunc("/{id}", router.delete).Methods("DELETE")
	s.HandleFunc("/deleteorg/{id}", router.completeOrgDeletion).Methods("POST")
	s.HandleFunc("/", router.create).Methods("POST")
	s.HandleFunc("/", router.getAll).Methods("GET")
}

func (router *OrganizationRouter) getDomainAccessibilityToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	domain := vars["domain"]
	if domain == "" {
		SendBadRequest(w)
		return
	}
	// Check if domain exists in activated state in ANY org already
	org, err := GetOrganizationRepository().GetOneByDomain(domain)
	if err != nil || org == nil {
		SendNotFound(w)
		return
	}
	res := &DomainAccessibilityPayload{
		Domain: domain,
		OrgID:  org.ID,
		Status: "ok",
	}
	SendJSON(w, res)
}

func (router *OrganizationRouter) getOrgForDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOneByDomain(vars["domain"])
	if e == nil || err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	res := &GetOrganizationResponse{}
	res.ID = e.ID
	res.Name = e.Name
	SendJSON(w, res)
}

func (router *OrganizationRouter) getOne(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, e.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	res := router.copyToRestModel(e)
	SendJSON(w, res)
}

func (router *OrganizationRouter) getDomains(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, e.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	list, err := GetOrganizationRepository().GetDomains(e)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := []*GetDomainResponse{}
	for _, domain := range list {
		item := &GetDomainResponse{
			DomainName:  domain.DomainName,
			Active:      domain.Active,
			VerifyToken: domain.VerifyToken,
			Primary:     domain.Primary,
			Accessible:  domain.Accessible,
			AccessCheck: domain.AccessCheck,
		}
		res = append(res, item)
	}
	SendJSON(w, res)
}

func (router *OrganizationRouter) addDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, e.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	featureCustomDomains, _ := GetSettingsRepository().GetBool(e.ID, SettingFeatureCustomDomains.Name)
	if !featureCustomDomains {
		SendPaymentRequired(w)
		return
	}
	domainName := strings.TrimSpace(strings.ToLower(vars["domain"]))
	if !ValidateDomain(domainName) {
		SendBadRequest(w)
		return
	}
	// Check if domain is special
	if strings.HasSuffix(domainName, ".seatsurfing.app") || strings.HasSuffix(domainName, ".seatsurfing.io") {
		SendBadRequest(w)
		return
	}
	// Check if domain exists in this org already
	domain, _ := GetOrganizationRepository().GetDomain(e, domainName)
	if domain != nil {
		SendAlreadyExists(w)
		return
	}
	// Check if domain exists in activated state in ANY org already
	someOrg, _ := GetOrganizationRepository().GetOneByDomain(domainName)
	if someOrg != nil {
		SendAlreadyExists(w)
		return
	}

	// when DOMAIN_VERIFICATION is off (self-hosted default) there is
	// no point in a TXT ownership challenge: the domain is active and accessible right away.
	domainVerification := GetConfig().DomainVerification
	err = GetOrganizationRepository().AddDomainWithAccessibility(e, domainName, !domainVerification, !domainVerification)
	if err != nil {
		log.Println(err)
		SendAlreadyExists(w)
		return
	}
	router.ensureOrgHasPrimaryDomain(e, domainName)
	SendCreated(w, domainName)
}

func (router *OrganizationRouter) verifyEmail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil || e == nil {
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, e.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	authState, err := GetAuthStateRepository().GetOneActive(vars["uuid"])
	if err != nil || authState == nil {
		SendNotFound(w)
		return
	}
	var m ChangeEmailAddressVerifyRequest
	if err := UnmarshalValidateBody(r, &m); err != nil {
		SendBadRequest(w)
		return
	}
	var authStatePayload ChangeOrgEmailPayload
	if err := json.Unmarshal([]byte(authState.Payload), &authStatePayload); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	if authStatePayload.OrgID != e.ID {
		log.Println("AuthState payload does not match organization ID")
		SendNotFound(w)
		return
	}
	if strconv.Itoa(authStatePayload.Code) != m.Code {
		log.Println("Invalid verification code")
		SendNotFound(w)
		return
	}
	e.ContactEmail = authStatePayload.Email
	if err := GetOrganizationRepository().Update(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *OrganizationRouter) verifyDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, e.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	domain, err := GetOrganizationRepository().GetDomain(e, vars["domain"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	if domain.Active {
		return
	}
	// Check if domain exists in activated state in ANY org already
	someOrg, _ := GetOrganizationRepository().GetOneByDomain(vars["domain"])
	if someOrg != nil {
		SendAlreadyExists(w)
		return
	}
	// With DOMAIN_VERIFICATION off, activate without a TXT ownership challenge.
	// This also covers domains added before the setting was turned off.
	if !GetConfig().DomainVerification {
		if err := GetOrganizationRepository().ActivateDomainAsAccessible(e, domain.DomainName); err != nil {
			log.Println(err)
			SendInternalServerError(w)
			return
		}
		SendUpdated(w)
		return
	}
	if !IsValidTXTRecord(domain.DomainName, domain.VerifyToken) {
		SendBadRequest(w)
		return
	}
	err = GetOrganizationRepository().ActivateDomain(e, domain.DomainName)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendUpdated(w)
}

func (router *OrganizationRouter) setPrimaryDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, e.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	domain, err := GetOrganizationRepository().GetDomain(e, vars["domain"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	if !domain.Active {
		SendBadRequest(w)
		return
	}
	GetOrganizationRepository().SetPrimaryDomain(e, vars["domain"])
	SendUpdated(w)
}

func (router *OrganizationRouter) removeDomain(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	org, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		log.Println(err)
		SendNotFound(w)
		return
	}
	user := GetRequestUser(r)
	if !(CanManagePlatform(user) || HasPermission(user, org.ID, PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	// prevent removing signup domain
	if strings.HasSuffix(vars["domain"], ".seatsurfing.app") {
		SendForbidden(w)
		return
	}
	domains, err := GetOrganizationRepository().GetDomains(org)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	if len(domains) <= 1 {
		SendBadRequest(w)
		return
	}
	err = GetOrganizationRepository().RemoveDomain(org, vars["domain"])
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	router.ensureOrgHasPrimaryDomain(org, "")
	SendUpdated(w)
}

func (router *OrganizationRouter) update(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	vars := mux.Vars(r)
	if !(CanManagePlatform(user) || HasPermission(user, vars["id"], PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	var m CreateOrganizationRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	if !isValidCreateOrganizationRequest(&m) {
		SendBadRequest(w)
		return
	}
	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	eIncoming := router.copyFromRestModel(&m)
	e.Name = eIncoming.Name
	e.Language = eIncoming.Language
	e.ContactFirstname = eIncoming.ContactFirstname
	e.ContactLastname = eIncoming.ContactLastname

	res := &ChangeOrgEmailResponse{
		VerifyUUID: "",
	}
	// Changing the contact address is confirmed by mail to the new address, so
	// that an organization cannot quietly be signed over to someone else. The
	// platform operator is exempt: they already hold the account, and the
	// confirmation would be addressed to a client they are acting for.
	if !CanManagePlatform(user) && !strings.EqualFold(e.ContactEmail, eIncoming.ContactEmail) {
		payload := &ChangeOrgEmailPayload{
			OrgID: e.ID,
			Email: eIncoming.ContactEmail,
			Code:  GetRandomNumber(100000, 999999), // Random 6-digit code
		}
		json, _ := json.Marshal(payload)
		authState := &AuthState{
			AuthStateType: AuthChangeOrgEmail,
			Payload:       string(json),
			Expiry:        time.Now().Add(time.Minute * 5),
		}
		if err := GetAuthStateRepository().Create(authState); err != nil {
			log.Println(err)
			SendInternalServerError(w)
			return
		}
		res.VerifyUUID = authState.ID
		if err := router.sendVerifyEmailAddressEmail(e, eIncoming.ContactEmail, strconv.Itoa(payload.Code)); err != nil {
			log.Println(err)
			SendInternalServerError(w)
			return
		}
	} else {
		e.ContactEmail = eIncoming.ContactEmail
	}
	if err := GetOrganizationRepository().Update(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendJSON(w, res)
}

func (router *OrganizationRouter) sendVerifyEmailAddressEmail(org *Organization, newEmail string, code string) error {
	vars := map[string]string{
		"recipientName":  org.ContactFirstname + " " + org.ContactLastname,
		"recipientEmail": newEmail,
		"code":           code,
	}
	return SendEmailWithOrg(&MailAddress{Address: newEmail}, GetEmailTemplatePathChangeEmailAddress(), org.Language, vars, org.ID)
}

func (router *OrganizationRouter) delete(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	vars := mux.Vars(r)

	if !(CanManagePlatform(user) || HasPermission(user, vars["id"], PermissionOrgSettings, PermissionLevelAdmin)) {
		SendForbidden(w)
		return
	}
	// The feature flag governs an organization deleting itself. It does not
	// bind the platform operator, for whom ending a client's workspace is part
	// of running the service.
	if !CanManagePlatform(user) && !GetConfig().AllowOrgDelete {
		SendForbidden(w)
		return
	}

	e, err := GetOrganizationRepository().GetOne(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}

	// The platform operator deletes a client's workspace outright. The emailed
	// confirmation code below exists for an organization deleting itself, where
	// the mail is the second factor; the operator already holds the account and
	// the mail would be addressed to the client they are acting for.
	if CanManagePlatform(user) {
		if GetUserRepository().IsPlatformOrganization(e.ID) {
			SendForbidden(w) // never delete the platform's own organization
			return
		}
		if err := GetOrganizationRepository().Delete(e); err != nil {
			log.Println(err)
			SendInternalServerError(w)
			return
		}
		SendUpdated(w)
		return
	}

	// send confirmation mail
	code := strconv.Itoa(GetRandomNumber(100000, 999999)) // Random 6-digit code
	payload := &AuthStateOrgDeletionRequestPayload{
		OrganizationID: e.ID,
		Code:           code,
	}
	authState := &AuthState{
		Expiry:        time.Now().Add(time.Hour * 1),
		AuthStateType: AuthDeleteOrg,
		Payload:       marshalAuthStateOrgDeletionRequestPayload(payload),
	}
	GetAuthStateRepository().Create(authState)
	if err := router.SendOrgConfirmDeleteOrgEmail(user, authState.ID, e); err != nil {
		log.Printf("Sending confirm org delete email failed: %s\n", err)
		SendInternalServerError(w)
		return
	}

	res := DeleteOrgResponse{
		Code: code,
	}
	SendJSON(w, res)
}

func marshalAuthStateOrgDeletionRequestPayload(payload *AuthStateOrgDeletionRequestPayload) string {
	json, _ := json.Marshal(payload)
	return string(json)
}

func unmarshalAuthStateOrgDeletionRequestPayload(payload string) *AuthStateOrgDeletionRequestPayload {
	var o *AuthStateOrgDeletionRequestPayload
	json.Unmarshal([]byte(payload), &o)
	return o
}

func (router *OrganizationRouter) completeOrgDeletion(w http.ResponseWriter, r *http.Request) {
	if !GetConfig().AllowOrgDelete {
		SendNotFound(w)
		return
	}
	var m CompleteOrgDeletionRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}

	// test auth state
	vars := mux.Vars(r)
	authState, err := GetAuthStateRepository().GetOneActive(vars["id"])
	if err != nil {
		SendNotFound(w)
		return
	}
	if authState.AuthStateType != AuthDeleteOrg {
		SendNotFound(w)
		return
	}
	payload := unmarshalAuthStateOrgDeletionRequestPayload(authState.Payload)
	if payload.Code != m.Code {
		SendNotFound(w)
		return
	}

	// (finally) delete organization
	organization, err := GetOrganizationRepository().GetOne(payload.OrganizationID)
	if organization == nil || err != nil {
		SendNotFound(w)
		return
	}
	GetOrganizationRepository().Delete(organization)

	SendUpdated(w)
}

func (router *OrganizationRouter) SendOrgConfirmDeleteOrgEmail(user *User, ID string, org *Organization) error {
	domain, err := GetOrganizationRepository().GetPrimaryDomain(org)
	if err != nil {
		return err
	}
	vars := map[string]string{
		"recipientName":  user.GetSafeRecipientName(),
		"recipientEmail": user.Email,
		"confirmID":      ID,
		"orgDomain":      FormatURL(domain.DomainName) + "/",
		"orgName":        org.Name,
	}
	language := org.Language
	if userLang, err := GetUserPreferencesRepository().Get(user.ID, PreferenceMailLanguage.Name); err == nil && userLang != "" {
		language = userLang
	}
	return SendEmailWithOrg(&MailAddress{Address: user.Email}, GetEmailTemplatePathConfirmDeleteOrg(), language, vars, org.ID)
}

// create registers a client organization. Only the platform operator does this
// directly; a client opens one of their own through createMyOrganization.
func (router *OrganizationRouter) create(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if !CanManagePlatform(user) {
		SendForbidden(w)
		return
	}
	var m CreateOrganizationRequest
	if UnmarshalValidateBody(r, &m) != nil {
		SendBadRequest(w)
		return
	}
	if !IsValidOrgName(m.Name) {
		SendBadRequest(w)
		return
	}
	e := router.copyFromRestModel(&m)
	e.SignupDate = time.Now()
	if err := GetOrganizationRepository().Create(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	GetRoleRepository().EnsureBuiltInRoles(e.ID)
	SendCreated(w, e.ID)
}

// getAll lists organizations. What it answers depends on who asks: the platform
// operator sees every client organization, and a client sees the ones they run.
func (router *OrganizationRouter) getAll(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if !CanManagePlatform(user) {
		// A client is a tenant of the platform, not an operator of it: the same
		// listing answers with the organizations they administer and stops there.
		router.getOwnOrganizations(w, user)
		return
	}
	list, err := GetOrganizationRepository().GetAll()
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	// The operator's own organization holds their account and the client
	// directory. It is not a workspace anyone bought, so it is left out of the
	// list of client organizations entirely.
	res := []*GetOrganizationListResponse{}
	for _, e := range list {
		if GetUserRepository().IsPlatformOrganization(e.ID) {
			continue
		}
		res = append(res, router.copyToListModel(e))
	}
	SendJSON(w, res)
}

// getOwnOrganizations lists the organizations a client administers.
//
// Membership is keyed on the email address: a client who runs several
// organizations holds one administrator record per organization, all sharing an
// email. The counts alongside each one are that client's own figures - they are
// read per organization and never span a tenant boundary.
func (router *OrganizationRouter) getOwnOrganizations(w http.ResponseWriter, user *User) {
	// Somebody who administers nothing has no organizations to be shown, and
	// an empty list would read as an answer rather than a refusal.
	if user == nil || !HasPermission(user, user.OrganizationID, PermissionOrgSettings, PermissionLevelAdmin) {
		SendForbidden(w)
		return
	}
	accounts, err := GetUserRepository().GetUsersWithEmail(user.Email)
	if err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	res := []*GetOrganizationListResponse{}
	for _, a := range accounts {
		if a.Disabled || !HasPermission(a, a.OrganizationID, PermissionOrgSettings, PermissionLevelAdmin) {
			continue
		}
		// The client's entry in the operator's directory is bookkeeping, not a
		// workspace they own, so it never appears among their organizations.
		if GetUserRepository().IsPlatformOrganization(a.OrganizationID) {
			continue
		}
		org, err := GetOrganizationRepository().GetOne(a.OrganizationID)
		if err != nil || org == nil {
			continue
		}
		res = append(res, router.copyToListModel(org))
	}
	sort.Slice(res, func(i, j int) bool {
		return strings.ToLower(res[i].Name) < strings.ToLower(res[j].Name)
	})
	SendJSON(w, res)
}

// copyToListModel adds the per-organization figures to the REST model.
func (router *OrganizationRouter) copyToListModel(e *Organization) *GetOrganizationListResponse {
	m := &GetOrganizationListResponse{
		GetOrganizationResponse: *router.copyToRestModel(e),
	}
	if num, err := GetUserRepository().GetCount(e.ID); err == nil {
		m.UserCount = num
	}
	if num, err := GetBookingRepository().GetCount(e.ID); err == nil {
		m.BookingCount = num
	}
	return m
}

// createMyOrganization lets a client open another workspace of their own.
//
// Every organization is a tenant in its own right: its own people, areas,
// groups and bookings, sharing nothing with the client's other organizations
// but the client's identity. That identity is what gets copied here - the same
// email and the same credential, as an administrator of the new organization -
// so the client signs in once and moves between their workspaces with the
// organization switcher.
func (router *OrganizationRouter) createMyOrganization(w http.ResponseWriter, r *http.Request) {
	user := GetRequestUser(r)
	if user == nil || !HasPermission(user, user.OrganizationID, PermissionOrgSettings, PermissionLevelAdmin) {
		SendForbidden(w)
		return
	}
	// The operator's organization is the client directory, not a workspace, and
	// the operator creates client organizations through the client record.
	if GetUserRepository().IsPlatformOrganization(user.OrganizationID) {
		SendForbidden(w)
		return
	}
	// Only the platform's own customer opens workspaces. A colleague the client
	// promoted to administrator runs the organization they were given; they are
	// not an account holder, and a workspace they created would belong to
	// nobody the operator has on file.
	if !IsPlatformClient(user) {
		SendForbidden(w)
		return
	}
	var m CreateMyOrganizationRequest
	if err := UnmarshalValidateBody(r, &m); err != nil {
		SendBadRequest(w)
		return
	}
	if !IsValidOrgName(m.Name) {
		SendBadRequest(w)
		return
	}
	current, err := GetOrganizationRepository().GetOne(user.OrganizationID)
	if err != nil || current == nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	e := &Organization{
		Name: m.Name,
		// The client runs this organization, so they are its contact. The
		// language follows the one they already have; it is not asked for again.
		ContactFirstname: user.Firstname,
		ContactLastname:  user.Lastname,
		ContactEmail:     user.Email,
		Language:         current.Language,
		SignupDate:       time.Now(),
	}
	if err := GetOrganizationRepository().Create(e); err != nil {
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	admin := &User{
		OrganizationID:         e.ID,
		Email:                  user.Email,
		Firstname:              user.Firstname,
		Lastname:               user.Lastname,
		HashedPassword:         user.HashedPassword,
		PasswordPending:        user.PasswordPending,
		PasswordUpdateRequired: user.PasswordUpdateRequired,
	}
	if err := GetUserRepository().Create(admin); err != nil {
		// A workspace nobody can administer is worse than no workspace at all,
		// so it does not survive a half-finished creation.
		GetOrganizationRepository().Delete(e)
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	orgAdminRoleID, _, _ := GetRoleRepository().EnsureBuiltInRoles(e.ID)
	if err := GetUserRoleRepository().Add(admin.ID, orgAdminRoleID, RoleAssignmentSourceManual); err != nil {
		GetOrganizationRepository().Delete(e)
		log.Println(err)
		SendInternalServerError(w)
		return
	}
	SendCreated(w, e.ID)
}

func (router *OrganizationRouter) ensureOrgHasPrimaryDomain(e *Organization, favoritePrimaryDomain string) {
	domains, _ := GetOrganizationRepository().GetDomains(e)
	hasPrimary := false
	for _, domain := range domains {
		if domain.Primary {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary {
		if favoritePrimaryDomain != "" {
			GetOrganizationRepository().SetPrimaryDomain(e, favoritePrimaryDomain)
		} else {
			domain, err := GetOrganizationRepository().GetPrimaryDomain(e)
			if err == nil && domain != nil {
				GetOrganizationRepository().SetPrimaryDomain(e, domain.DomainName)
			}
		}
	}
}

func isValidCreateOrganizationRequest(m *CreateOrganizationRequest) bool {
	return IsValidOrgName(m.Name) && ValidateEmail(m.Email) && IsValidHumanName(m.Firstname) && IsValidHumanName(m.Lastname) && IsValidOrgLanguage(m.Language)
}

func (router *OrganizationRouter) copyFromRestModel(m *CreateOrganizationRequest) *Organization {
	e := &Organization{}
	e.Name = m.Name
	e.ContactFirstname = m.Firstname
	e.ContactLastname = m.Lastname
	e.ContactEmail = m.Email
	e.Language = m.Language
	return e
}

func (router *OrganizationRouter) copyToRestModel(e *Organization) *GetOrganizationResponse {
	m := &GetOrganizationResponse{}
	m.ID = e.ID
	m.Name = e.Name
	m.Firstname = e.ContactFirstname
	m.Lastname = e.ContactLastname
	m.Email = e.ContactEmail
	m.Language = e.Language
	return m
}
