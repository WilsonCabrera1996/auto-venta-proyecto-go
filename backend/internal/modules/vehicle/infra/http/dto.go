package http

// vehicleRequest es el cuerpo de alta/edicion que acepta HTTP.
// Price se expone como string decimal ("45000.00") y se guarda en centavos para
// evitar errores de punto flotante; la conversion vive aqui, no en el dominio.
type vehicleRequest struct {
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Year         int    `json:"year"`
	Price        string `json:"price"`
	Status       string `json:"status,omitempty"`
	VIN          string `json:"vin,omitempty"`
	Plate        string `json:"plate,omitempty"`
	Mileage      int64  `json:"mileage"`
	Color        string `json:"color,omitempty"`
	FuelType     string `json:"fuel_type,omitempty"`
	Transmission string `json:"transmission,omitempty"`
	BodyType     string `json:"body_type,omitempty"`
	Doors        int    `json:"doors,omitempty"`
	Seats        int    `json:"seats,omitempty"`
	EngineCC     int    `json:"engine_cc,omitempty"`
	Horsepower   int    `json:"horsepower,omitempty"`
	Drivetrain   string `json:"drivetrain,omitempty"`
	Description  string `json:"description,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
}

// vehicleResponse es la representacion publica del recurso. No expone la entidad
// cruda: el contrato HTTP puede evolucionar sin arrastrar al dominio.
type vehicleResponse struct {
	ID           string `json:"id"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Year         int    `json:"year"`
	Price        string `json:"price"`
	PriceCents   int64  `json:"price_cents"`
	Status       string `json:"status"`
	VIN          string `json:"vin,omitempty"`
	Plate        string `json:"plate,omitempty"`
	Mileage      int64  `json:"mileage"`
	Color        string `json:"color,omitempty"`
	FuelType     string `json:"fuel_type,omitempty"`
	Transmission string `json:"transmission,omitempty"`
	BodyType     string `json:"body_type,omitempty"`
	Doors        int    `json:"doors,omitempty"`
	Seats        int    `json:"seats,omitempty"`
	EngineCC     int    `json:"engine_cc,omitempty"`
	Horsepower   int    `json:"horsepower,omitempty"`
	Drivetrain   string `json:"drivetrain,omitempty"`
	Description  string `json:"description,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type statusRequest struct {
	Status string `json:"status"`
}
