package data

type User struct {
	ID        string    `json:"id" binding:"required"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Interests []string  `json:"interest"`
	Hobbies   []string  `json:"hobby"`
	Age       int       `json:"age"`
	Address   Address   `json:"address"`
	Location  *GeoPoint `json:"location,omitempty"`
	Gender    string    `json:"gender"`
	Status    string    `json:"status"`
	Photo     string    `json:"photo"`
}

// GeoPoint is serialized in the object format Elasticsearch expects for geo_point fields.
type GeoPoint struct {
	Lat float64 `json:"lat" binding:"min=-90,max=90"`
	Lon float64 `json:"lon" binding:"min=-180,max=180"`
}

type UserServiceRequest struct {
	ReferenceID string    `json:"referenceId"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Password    string    `json:"password"`
	Interests   []string  `json:"interests"`
	Hobbies     []string  `json:"hobbies"`
	Age         int       `json:"age"`
	Address     Address   `json:"address"`
	Location    *GeoPoint `json:"location,omitempty"`
	Gender      string    `json:"gender"`
	Status      string    `json:"status"`
	Photo       string    `json:"photo"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}
