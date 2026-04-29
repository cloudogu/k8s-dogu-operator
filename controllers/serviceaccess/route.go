package serviceaccess

type Route struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Location string `json:"location"`
	Pass     string `json:"pass"`
	Rewrite  string `json:"rewrite,omitempty"`
}
