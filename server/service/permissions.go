package service

import (
	"log"

	. "github.com/seatsurfing/seatsurfing/server/api"
	. "github.com/seatsurfing/seatsurfing/server/repository"
)

// GetEffectivePermissions resolves the access a user has within an
// organization: for each permission, the highest level granted by any of their
// assigned roles. A user is never granted anything outside their own
// organization.
//
// Permissions absent from the result are not granted. Note that this covers
// only administrative functionality: every authenticated user additionally has
// a fixed baseline (their own bookings, buddies, preferences, profile and MFA,
// and read access to locations, spaces and availability) which is not
// represented here and can not be revoked.
func GetEffectivePermissions(user *User, organizationID string) map[Permission]PermissionLevel {
	if user == nil || user.OrganizationID != organizationID {
		return map[Permission]PermissionLevel{}
	}
	perms, err := GetUserRoleRepository().GetEffectivePermissions(user.ID)
	if err != nil {
		// Fail closed: an unreadable assignment must not grant access.
		log.Println(err)
		return map[Permission]PermissionLevel{}
	}
	return perms
}

// HasPermission reports whether the user holds at least the given level for a
// permission within the organization.
func HasPermission(user *User, organizationID string, p Permission, min PermissionLevel) bool {
	if min <= PermissionLevelNone {
		return true
	}
	return GetEffectivePermissions(user, organizationID)[p] >= min
}

// HasAnyPermission reports whether the user holds any administrative
// permission at all within the organization. It backs the checks that used to
// ask "is this user some kind of admin", such as whether to show the link into
// the administration UI or whether an admins-only MFA policy applies.
func HasAnyPermission(user *User, organizationID string) bool {
	return len(GetEffectivePermissions(user, organizationID)) > 0
}

// CanAccessOrg reports organization membership. It is not a privilege check:
// every authenticated user has baseline access to their own organization.
func CanAccessOrg(user *User, organizationID string) bool {
	return user != nil && user.OrganizationID == organizationID
}

// CanManagePlatform reports whether the user runs the SaaS platform itself.
//
// Platform power is deliberately account-scoped, never data-scoped: it governs
// the lifecycle of client organizations (create, rename, domains, plan,
// suspend, delete) and nothing inside them. The operator must not be able to
// read a client's users, bookings, areas or analytics, so this must never be
// used to satisfy an authorization check on tenant data - use CanAccessOrg or
// HasPermission for that, neither of which grants the operator anything
// outside their own organization.
func CanManagePlatform(user *User) bool {
	if user == nil {
		return false
	}
	return HasPermission(user, user.OrganizationID, PermissionPlatform, PermissionLevelAdmin)
}

// IsPlatformClient reports whether this identity is one of the platform's
// customers, which is to say whether it holds an entry in the operator's client
// directory - the operator's own organization.
//
// The distinction is between the account holder and the people inside their
// business. A client may promote a colleague to administrator of an
// organization; that colleague runs the workspace they were given, but they
// are not the customer, and nothing they open would answer to an account the
// operator has on file.
func IsPlatformClient(user *User) bool {
	if user == nil {
		return false
	}
	accounts, err := GetUserRepository().GetUsersWithEmail(user.Email)
	if err != nil {
		log.Println(err)
		return false
	}
	for _, a := range accounts {
		if GetUserRepository().IsPlatformOrganization(a.OrganizationID) {
			return true
		}
	}
	return false
}

// hasNoAdminRestrictions reports whether the organization lifts booking
// restrictions for booking admins and user is one.
func hasNoAdminRestrictions(user *User, organizationID string) bool {
	noAdminRestrictions, _ := GetSettingsRepository().GetBool(organizationID, SettingNoAdminRestrictions.Name)
	return noAdminRestrictions && HasPermission(user, organizationID, PermissionBookings, PermissionLevelAdmin)
}
