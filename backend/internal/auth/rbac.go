package auth

import "fmt"

// Permission is a bitflag representing a single authorization action.
// Up to 64 permissions can be defined in a single uint64.
type Permission uint64

const (
	PermViewDashboard Permission = 1 << iota
	PermManageUsers
	PermManageInbounds
	PermManageOutbounds
	PermManageCores
	PermManageSettings
	PermViewLogs
	PermManageCertificates
	PermManageBackups
	PermSuperAdmin
	PermManageWarp
	PermManageGeo
	PermManageNotifications
)

func (p Permission) String() string {
	switch p {
	case PermViewDashboard:
		return "view:dashboard"
	case PermManageUsers:
		return "manage:users"
	case PermManageInbounds:
		return "manage:inbounds"
	case PermManageOutbounds:
		return "manage:outbounds"
	case PermManageCores:
		return "manage:cores"
	case PermManageSettings:
		return "manage:settings"
	case PermViewLogs:
		return "view:logs"
	case PermManageCertificates:
		return "manage:certificates"
	case PermManageBackups:
		return "manage:backups"
	case PermSuperAdmin:
		return "superadmin"
	case PermManageWarp:
		return "manage:warp"
	case PermManageGeo:
		return "manage:geo"
	case PermManageNotifications:
		return "manage:notifications"
	default:
		return fmt.Sprintf("permission:%d", p)
	}
}

// Permissions is a bitset of granted permissions.
type Permissions uint64

func NewPermissions(perms ...Permission) Permissions {
	var p Permissions
	for _, perm := range perms {
		p |= Permissions(perm)
	}
	return p
}

func (p Permissions) Has(perm Permission) bool {
	return p&Permissions(perm) != 0
}

func (p Permissions) HasAll(perms ...Permission) bool {
	for _, perm := range perms {
		if !p.Has(perm) {
			return false
		}
	}
	return true
}

func (p Permissions) HasAny(perms ...Permission) bool {
	for _, perm := range perms {
		if p.Has(perm) {
			return true
		}
	}
	return false
}

func (p Permissions) Grant(perms ...Permission) Permissions {
	for _, perm := range perms {
		p |= Permissions(perm)
	}
	return p
}

func (p Permissions) Revoke(perms ...Permission) Permissions {
	for _, perm := range perms {
		p &^= Permissions(perm)
	}
	return p
}
