package data

type User struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Interest []string `json:"interest"`
	Hobby    []string `json:"hobby"`
	Age      int      `json:"age"`
	Address  Address  `json:"address"`
	Gender   string   `json:"gender"`
	Status   string   `json:"status"`
	Photo    string   `json:"photo"`
}

type UserServiceRequest struct {
	ReferenceID string   `json:"referenceId"`
	Name        string   `json:"name"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Interests   []string `json:"interests"`
	Hobbies     []string `json:"hobbies"`
	Age         int      `json:"age"`
	Address     Address  `json:"address"`
	Gender      string   `json:"gender"`
	Status      string   `json:"status"`
	Photo       string   `json:"photo"`
}

type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Zip     string `json:"zip"`
	Country string `json:"country"`
}
