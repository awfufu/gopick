package domain

type Order struct {
	ID                    string         `json:"id"`
	ShopID                string         `json:"shopId,omitempty"`
	LogisticID            string         `json:"logisticId,omitempty"`
	City                  int            `json:"city"`
	Platform              string         `json:"platform"`
	DailyPlatformSequence int            `json:"dailyPlatformSequence"`
	OrderNo               string         `json:"orderNo"`
	OrderTime             string         `json:"orderTime"`
	UserAddress           string         `json:"userAddress"`
	RawShopName           string         `json:"rawShopName,omitempty"`
	ShopAddress           string         `json:"shopAddress,omitempty"`
	RawShopAddress        string         `json:"rawShopAddress,omitempty"`
	IsSubscribe           bool           `json:"isSubscribe,omitempty"`
	CompletedAt           string         `json:"completedAt,omitempty"`
	Longitude             float64        `json:"longitude"`
	Latitude              float64        `json:"latitude"`
	Status                string         `json:"status,omitempty"`
	DeliveryDeadline      string         `json:"deliveryDeadline,omitempty"`
	DeliveryTimeRange     string         `json:"deliveryTimeRange,omitempty"`
	DistanceKM            float64        `json:"distanceKm"`
	DistanceIsLinear      bool           `json:"distanceIsLinear"`
	ActualPaid            int            `json:"actualPaid"`
	ExpectedIncome        int            `json:"expectedIncome,omitempty"`
	PlatformCommission    int            `json:"platformCommission"`
	Delivery              *Delivery      `json:"delivery,omitempty"`
	Items                 []OrderItem    `json:"items"`
	Raw                   map[string]any `json:"raw,omitempty"`
}

type OrderItem struct {
	ProductName string `json:"productName"`
	ProductNo   string `json:"productNo,omitempty"`
	Quantity    int    `json:"quantity"`
	Thumb       string `json:"thumb,omitempty"`
}

type Delivery struct {
	LogisticName     string `json:"logisticName,omitempty"`
	SendFee          int    `json:"sendFee,omitempty"`
	Tip              int    `json:"tip,omitempty"`
	PremiumFee       int    `json:"premiumFee,omitempty"`
	TotalDeliveryFee int    `json:"totalDeliveryFee,omitempty"`
	PickupTime       string `json:"pickupTime,omitempty"`
	Track            string `json:"track,omitempty"`
	RiderName        string `json:"riderName,omitempty"`
}

type OrderContext struct {
	MerchantID string `json:"merchantId"`
	AccountID  string `json:"accountId"`
}

type OrderStatus string

const (
	OrderStatusConfirm    OrderStatus = "confirm"
	OrderStatusSubscribe  OrderStatus = "subscribe"
	OrderStatusDelivery   OrderStatus = "delivery"
	OrderStatusPickup     OrderStatus = "pickup"
	OrderStatusDelivering OrderStatus = "delivering"
	OrderStatusExpect     OrderStatus = "expect"
	OrderStatusCancel     OrderStatus = "cancel"
	OrderStatusRemind     OrderStatus = "remind"
	OrderStatusMeal       OrderStatus = "meal"
)

func AllowedOrderStatuses() []OrderStatus {
	return []OrderStatus{
		OrderStatusConfirm,
		OrderStatusSubscribe,
		OrderStatusDelivery,
		OrderStatusPickup,
		OrderStatusDelivering,
		OrderStatusExpect,
		OrderStatusCancel,
		OrderStatusRemind,
		OrderStatusMeal,
	}
}

func IsAllowedOrderStatus(status OrderStatus) bool {
	for _, candidate := range AllowedOrderStatuses() {
		if status == candidate {
			return true
		}
	}

	return false
}
