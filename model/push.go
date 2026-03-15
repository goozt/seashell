package model

// PushSubscription is the Web Push subscription sent by the browser.
type PushSubscription struct {
	UserID   string               `json:"user_id"`
	Endpoint string               `json:"endpoint"`
	Keys     PushSubscriptionKeys `json:"keys"`
}

type PushSubscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// PushPayload is the JSON body delivered inside a Web Push message.
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url,omitempty"`
	Tag   string `json:"tag,omitempty"`
}
