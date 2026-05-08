package domain

type WSStatus struct {
	Connected           bool   `json:"connected"`
	Authenticated       bool   `json:"authenticated"`
	URL                 string `json:"url"`
	MerchantID          string `json:"merchantId,omitempty"`
	AccountID           string `json:"accountId,omitempty"`
	LastMessageCmd      string `json:"lastMessageCmd,omitempty"`
	LastMessageAt       string `json:"lastMessageAt,omitempty"`
	LastHeartbeatAt     string `json:"lastHeartbeatAt,omitempty"`
	LastConnectedAt     string `json:"lastConnectedAt,omitempty"`
	LastAuthenticatedAt string `json:"lastAuthenticatedAt,omitempty"`
	ReconnectCount      int    `json:"reconnectCount"`
	LastError           string `json:"lastError,omitempty"`
}
