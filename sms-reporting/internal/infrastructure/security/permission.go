package security

type PermissionCode string

const (
	PermServerCreate PermissionCode = "server:create"
	PermServerRead   PermissionCode = "server:read"
	PermServerUpdate PermissionCode = "server:update"
	PermServerDelete PermissionCode = "server:delete"
	PermServerImport PermissionCode = "server:import"
	PermServerExport PermissionCode = "server:export"

	PermReportRequest PermissionCode = "report:request"
)
