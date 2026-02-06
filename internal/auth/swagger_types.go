package auth

// LoginRequest is the request body for POST /auth/login.
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"securepassword123"`
}

// RefreshRequest is the request body for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJl..."`
}

// LogoutRequest is the request body for POST /auth/logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" example:"dGhpcyBpcyBhIHJlZnJl..."`
}

// SetupRequest is the request body for POST /auth/setup.
type SetupRequest struct {
	Username string `json:"username" example:"admin"`
	Email    string `json:"email" example:"admin@example.com"`
	Password string `json:"password" example:"securepassword123"`
}

// UpdateUserRequest is the request body for PUT /users/{id}.
type UpdateUserRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Role     string `json:"role" example:"operator"`
	Disabled bool   `json:"disabled" example:"false"`
}
