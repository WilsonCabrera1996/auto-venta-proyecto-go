package http

// loginRequest son las credenciales. El password entra en claro por el cable
// y por eso el servicio solo puede existir detrás de TLS: en Render eso es
// automatico (HTTPS en el borde), pero en local no jugar con HTTP plano.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type crearUsuarioRequest struct {
	Email    string `json:"email"`
	Nombre   string `json:"nombre"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type cambiarRolRequest struct {
	Role string `json:"role"`
}

// userResponse nunca incluye PasswordHash: si se filtrara ahi, el hash viaja al
// navegador en cada listado de usuarios.
type userResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Nombre    string   `json:"nombre"`
	Role      string   `json:"role"`
	Activo    bool     `json:"activo"`
	Permisos  []string `json:"permisos"`
	CreatedAt string   `json:"created_at"`
}

// loginResponse entrega el token en claro. Es el unico lugar del sistema donde
// eso ocurre.
type loginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt string       `json:"expires_at"`
	User      userResponse `json:"user"`
}
