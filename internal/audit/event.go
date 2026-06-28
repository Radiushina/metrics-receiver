package audit

type Event struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IpAddress string   `json:"ip_address"`
}
