package security

import (
	"context"
)

type Authorizer struct {
	rolePermissions map[string]map[PermissionCode]bool
}

func NewAuthorizer() *Authorizer {
	// Static RBAC mapping
	adminPerms := map[PermissionCode]bool{
		PermServerCreate:  true,
		PermServerRead:    true,
		PermServerUpdate:  true,
		PermServerDelete:  true,
		PermServerImport:  true,
		PermServerExport:  true,
		PermReportRequest: true,
	}

	userPerms := map[PermissionCode]bool{
		PermServerRead: true, // Only allow viewing servers
	}

	return &Authorizer{
		rolePermissions: map[string]map[PermissionCode]bool{
			"ADMIN": adminPerms,
			"USER":  userPerms,
		},
	}
}

func (a *Authorizer) HasPermission(ctx context.Context, principal *Principal, requiredPermission PermissionCode) bool {
	if requiredPermission == "" {
		return true // Allow if no specific permission is required
	}

	if principal == nil {
		return false
	}

	// Always grant if ADMIN
	if principal.RoleCode == "ADMIN" {
		return true
	}

	perms, ok := a.rolePermissions[principal.RoleCode]
	if !ok {
		return false
	}

	return perms[requiredPermission]
}
