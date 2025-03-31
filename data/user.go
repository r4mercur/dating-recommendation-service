package data

type User struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Interest []string `json:"interest"`
	Hobby    []string `json:"hobby"`
	Age      int      `json:"age"`
	Address  string   `json:"address"`
	Gender   string   `json:"gender"`
	Status   string   `json:"status"`
	Photo    string   `json:"photo"`
}
