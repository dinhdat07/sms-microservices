package security

type Principal struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	RoleID      string `json:"role_id"`
	RoleCode    string `json:"role_code"`
	SessionID   string `json:"session_id"`
}
