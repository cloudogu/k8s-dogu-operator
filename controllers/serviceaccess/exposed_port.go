package serviceaccess

type ExposedPort struct {
	Protocol   string `json:"protocol"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort"`
}
