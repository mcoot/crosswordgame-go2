package request

// CreateGuestRequest is the request body for creating a guest player
type CreateGuestRequest struct {
	DisplayName string `json:"display_name"`
}

// RegisterRequest is the request body for registering a player
type RegisterRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// LoginRequest is the request body for logging in
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateLobbyRequest is the request body for creating a lobby
type CreateLobbyRequest struct {
	GridSize int `json:"grid_size,omitempty"`
}

// UpdateConfigRequest is the request body for updating lobby config
type UpdateConfigRequest struct {
	GridSize int `json:"grid_size"`
}

// SetRoleRequest is the request body for setting a member's role
type SetRoleRequest struct {
	Role string `json:"role"`
}

// TransferHostRequest is the request body for transferring host
type TransferHostRequest struct {
	NewHostID string `json:"new_host_id"`
}

// AnnounceRequest is the request body for announcing a letter
type AnnounceRequest struct {
	Letter string `json:"letter"`
}

// PlaceRequest is the request body for placing a letter
type PlaceRequest struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

// AddBotRequest is the request body for adding a bot to a lobby
type AddBotRequest struct {
	DisplayName string `json:"display_name,omitempty"`
}
