package maiyatian

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/awfufu/gopick/internal/config"
	"github.com/awfufu/gopick/internal/domain"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
	cookie  string
	ua      string
}

const maiyatianBaseURL = "https://saas.maiyatian.com"

type rawListResponse struct {
	Errno   int            `json:"errno"`
	Message string         `json:"message"`
	Data    []rawListOrder `json:"data"`
}

type rawQueryListResponse struct {
	Errno   int              `json:"errno"`
	Message string           `json:"message"`
	Data    []map[string]any `json:"data"`
}

type rawDetailResponse struct {
	Errno   int            `json:"errno"`
	Message string         `json:"message"`
	Data    rawDetailOrder `json:"data"`
}

type rawListOrder struct {
	ID               string         `json:"id"`
	SourceID         string         `json:"source_id"`
	MerchantID       string         `json:"merchant_id"`
	City             string         `json:"city"`
	ShopID           string         `json:"shop_id"`
	SourceSN         string         `json:"source_sn"`
	StatusStr        string         `json:"status_str"`
	IsSubscribe      string         `json:"is_subscribe"`
	MapAddress       string         `json:"map_address"`
	Address          string         `json:"address"`
	TotalPrice       any            `json:"total_price"`
	BalancePrice     any            `json:"balance_price"`
	OrderTime        any            `json:"order_time"`
	DeliveryTime     any            `json:"delivery_time"`
	FinishedTime     any            `json:"finished_time"`
	DeliveryID       any            `json:"delivery_id"`
	Longitude        any            `json:"longitude"`
	Latitude         any            `json:"latitude"`
	DeliveryDistance any            `json:"delivery_distance"`
	Tips             string         `json:"tips"`
	ChannelTagName   string         `json:"channel_tag_name"`
	Extend           rawOrderExtend `json:"extend"`
	Shop             rawOrderShop   `json:"shop"`
	Delivery         rawDelivery    `json:"delivery"`
}

type rawOrderExtend struct {
	ChannelName string `json:"channel_name"`
}

type rawOrderShop struct {
	Name      string `json:"name"`
	Longitude string `json:"longitude"`
	Latitude  string `json:"latitude"`
}

type rawDelivery struct {
	LogisticID   any    `json:"logistic_id"`
	LogisticName string `json:"logistic_name"`
	DeliveryName string `json:"delivery_name"`
	Track        string `json:"track"`
	SendFee      any    `json:"send_fee"`
	Tip          any    `json:"tip"`
	PremiumFee   any    `json:"premium_fee"`
	PickupTime   any    `json:"pickup_time"`
}

type rawDetailOrder struct {
	ID                 any              `json:"id"`
	SourceID           any              `json:"source_id"`
	MerchantID         any              `json:"merchant_id"`
	City               any              `json:"city"`
	ShopID             any              `json:"shop_id"`
	SourceSN           any              `json:"source_sn"`
	ChannelTagName     string           `json:"channel_tag_name"`
	MapAddress         string           `json:"map_address"`
	Address            string           `json:"address"`
	OrderTime          any              `json:"order_time"`
	DeliveryTime       any              `json:"delivery_time"`
	DeliveryTimeFormat string           `json:"delivery_time_format"`
	FinishedTime       any              `json:"finished_time"`
	Longitude          any              `json:"longitude"`
	Latitude           any              `json:"latitude"`
	DeliveryDistance   any              `json:"delivery_distance"`
	Tips               string           `json:"tips"`
	IsSubscribe        any              `json:"is_subscribe"`
	Extend             rawOrderExtend   `json:"extend"`
	Delivery           rawDelivery      `json:"delivery"`
	Fee                rawDetailFee     `json:"fee"`
	Goods              []rawDetailGoods `json:"goods"`
	Shop               rawOrderShop     `json:"shop"`
}

type rawDetailFee struct {
	UserFee    any `json:"user_fee"`
	ShopFee    any `json:"shop_fee"`
	Commission any `json:"commission"`
}

type rawDetailGoods struct {
	GoodsName string `json:"goods_name"`
	SKUCode   string `json:"sku_code"`
	Number    any    `json:"number"`
	Thumb     string `json:"thumb"`
}

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)
var merchantIDPattern = regexp.MustCompile(`var\s+gMerchantId\s*=\s*'([^']+)'`)
var accountIDPattern = regexp.MustCompile(`var\s+gAccountId\s*=\s*'([^']+)'`)

func NewHTTPClient(cfg config.MaiyatianConfig) *HTTPClient {
	return &HTTPClient{
		baseURL: maiyatianBaseURL,
		cookie:  cfg.Cookie,
		ua:      cfg.UserAgent,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *HTTPClient) ListOrders(ctx context.Context, status domain.OrderStatus) ([]domain.Order, error) {
	if !isAllowedStatus(status) {
		return nil, fmt.Errorf("unsupported status: %s", status)
	}

	endpoint := fmt.Sprintf("%s/order/list/?%s", c.baseURL, buildListOrdersQuery(status).Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	applyDefaultHeaders(req, c)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("maiyatian list orders returned status %d", resp.StatusCode)
	}

	var payload rawListResponse
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode maiyatian list orders: %w", err)
	}
	if payload.Errno != 1 {
		return nil, fmt.Errorf("maiyatian list orders errno=%d message=%s", payload.Errno, strings.TrimSpace(payload.Message))
	}

	orders := make([]domain.Order, 0, len(payload.Data))
	for _, item := range payload.Data {
		order := mapRawListOrder(item)
		if order.OrderNo == "" {
			continue
		}
		if err := c.enrichOrderWithDetail(ctx, &order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (c *HTTPClient) GetOrderContext(ctx context.Context) (domain.OrderContext, error) {
	endpoint := fmt.Sprintf("%s/order/", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.OrderContext{}, err
	}

	applyOrderPageHeaders(req, c)

	resp, err := c.client.Do(req)
	if err != nil {
		return domain.OrderContext{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return domain.OrderContext{}, fmt.Errorf("maiyatian order page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.OrderContext{}, fmt.Errorf("read maiyatian order page: %w", err)
	}

	html := string(body)
	merchantID := extractFirstSubmatch(merchantIDPattern, html)
	accountID := extractFirstSubmatch(accountIDPattern, html)
	if merchantID == "" || accountID == "" {
		return domain.OrderContext{}, fmt.Errorf("failed to extract merchant/account id from maiyatian order page")
	}

	return domain.OrderContext{
		MerchantID: merchantID,
		AccountID:  accountID,
	}, nil
}

func (c *HTTPClient) GetOrderByID(ctx context.Context, id string) (domain.Order, error) {
	detail, err := c.fetchOrderDetail(ctx, id)
	if err != nil {
		return domain.Order{}, err
	}

	platform := normalizePlatformName(detail.ChannelTagName)
	shopAddress := readPreferredShopAddressFromDetail(detail)
	order := domain.Order{
		ID:                    strings.TrimSpace(asString(detail.ID)),
		ShopID:                firstNonEmptyString(strings.TrimSpace(asString(detail.ShopID)), strings.TrimSpace(asString(detail.MerchantID))),
		City:                  maxInt(0, parseInt(detail.City)),
		Platform:              platform,
		DailyPlatformSequence: parseInt(detail.SourceSN),
		OrderNo:               strings.TrimSpace(asString(detail.SourceID)),
		OrderTime:             parseOrderTime(detail.OrderTime),
		UserAddress:           firstNonEmptyString(detail.MapAddress, detail.Address),
		RawShopName:           readPreferredShopNameFromDetail(detail),
		ShopAddress:           shopAddress,
		RawShopAddress:        shopAddress,
		IsSubscribe:           parseBool(detail.IsSubscribe),
		CompletedAt:           formatUnixTimestamp(detail.FinishedTime),
		Longitude:             parseFloat(detail.Longitude),
		Latitude:              parseFloat(detail.Latitude),
		Status:                stripHTML(detail.Tips),
		DeliveryDeadline:      formatDeadline(detail.DeliveryTime),
		DeliveryTimeRange:     strings.TrimSpace(detail.DeliveryTimeFormat),
		DistanceKM:            maxFloat(0, parseFloat(detail.DeliveryDistance)/1000),
		DistanceIsLinear:      false,
		Items:                 []domain.OrderItem{},
	}

	applyDetailToOrder(&order, detail)
	return order, nil
}

func (c *HTTPClient) ListAllOrders(ctx context.Context, date string) ([]domain.Order, error) {
	normalizedDate := strings.TrimSpace(date)
	if normalizedDate == "" {
		normalizedDate = time.Now().Format("2006-01-02")
	}

	orders := make([]domain.Order, 0, 64)
	for page := 1; page <= 200; page++ {
		payload, err := c.fetchAllOrdersPage(ctx, normalizedDate, page)
		if err != nil {
			return nil, err
		}

		for _, item := range payload.Data {
			order := mapRawQueryOrder(item)
			if order.OrderNo == "" {
				continue
			}
			if err := c.enrichOrderWithDetail(ctx, &order); err != nil {
				return nil, err
			}
			orders = append(orders, order)
		}

		if len(payload.Data) < 20 {
			break
		}
	}

	return orders, nil
}

func (c *HTTPClient) fetchAllOrdersPage(ctx context.Context, date string, page int) (rawQueryListResponse, error) {
	endpoint := fmt.Sprintf("%s/query/list/?%s", c.baseURL, buildAllOrdersQuery(date, page).Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return rawQueryListResponse{}, err
	}

	applyDefaultHeaders(req, c)

	resp, err := c.client.Do(req)
	if err != nil {
		return rawQueryListResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return rawQueryListResponse{}, fmt.Errorf("maiyatian all orders returned status %d", resp.StatusCode)
	}

	var payload rawQueryListResponse
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return rawQueryListResponse{}, fmt.Errorf("decode maiyatian all orders: %w", err)
	}
	if payload.Errno != 1 {
		return rawQueryListResponse{}, fmt.Errorf("maiyatian all orders errno=%d message=%s", payload.Errno, strings.TrimSpace(payload.Message))
	}

	return payload, nil
}

func (c *HTTPClient) fetchOrderDetail(ctx context.Context, orderID string) (rawDetailOrder, error) {
	endpoint := fmt.Sprintf("%s/order/detail/?detail=1&f=json&id=%s", c.baseURL, url.QueryEscape(orderID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return rawDetailOrder{}, err
	}

	applyDefaultHeaders(req, c)

	resp, err := c.client.Do(req)
	if err != nil {
		return rawDetailOrder{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return rawDetailOrder{}, fmt.Errorf("maiyatian order detail returned status %d", resp.StatusCode)
	}

	var payload rawDetailResponse
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return rawDetailOrder{}, fmt.Errorf("decode maiyatian order detail: %w", err)
	}
	if payload.Errno != 1 {
		return rawDetailOrder{}, fmt.Errorf("maiyatian order detail errno=%d message=%s", payload.Errno, strings.TrimSpace(payload.Message))
	}

	return payload.Data, nil
}

func (c *HTTPClient) enrichOrderWithDetail(ctx context.Context, order *domain.Order) error {
	if order == nil {
		return nil
	}

	orderID := strings.TrimSpace(order.ID)
	if orderID == "" {
		return nil
	}

	detail, err := c.fetchOrderDetail(ctx, orderID)
	if err != nil {
		return err
	}

	applyDetailToOrder(order, detail)
	return nil
}

func buildListOrdersQuery(status domain.OrderStatus) url.Values {
	query := url.Values{}
	query.Set("page", "1")
	query.Set("status", string(status))
	query.Set("is_sort", "0")
	query.Set("page_size", "20")
	query.Set("sort", "1")
	query.Set("shop_id", "undefined")
	query.Set("delivery_type", "0")
	query.Set("dispatch_status", "0")
	query.Set("meal_status", "0")
	query.Set("f", "json")
	return query
}

func buildAllOrdersQuery(date string, page int) url.Values {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("page_size", "20")
	query.Set("filter_type", "all")
	query.Set("filter_goods_num", "goods_number")
	query.Set("filter_gird", "all")
	query.Set("filter_label", "all")
	query.Set("filter_time", "0")
	query.Set("filter_stime", "60")
	query.Set("filter_date", date)
	query.Set("date_type", "order_date")
	query.Set("shop_id", "0")
	query.Set("mode", "list")
	query.Set("sort_map", "[object Object]")
	query.Set("controller", "open")
	query.Set("sort", "1")
	query.Set("goods_number", "0")
	query.Set("f", "json")
	return query
}

func applyDefaultHeaders(req *http.Request, client *HTTPClient) {
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", client.baseURL+"/order/")
	req.Header.Set("M-Appkey", "fe_com.sankuai.dap.mytstats")
	req.Header.Set("User-Agent", client.ua)
	if client.cookie != "" {
		req.Header.Set("Cookie", client.cookie)
	}
}

func applyOrderPageHeaders(req *http.Request, client *HTTPClient) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Referer", client.baseURL+"/query/")
	req.Header.Set("User-Agent", client.ua)
	if client.cookie != "" {
		req.Header.Set("Cookie", client.cookie)
	}
}

func mapRawListOrder(raw rawListOrder) domain.Order {
	platform := normalizePlatformName(raw.ChannelTagName)
	actualPaid := parseInt(raw.TotalPrice)
	expectedIncome := parseInt(raw.BalancePrice)
	logisticID := firstNonEmptyString(asString(raw.Delivery.LogisticID), asString(raw.DeliveryID))

	return domain.Order{
		ID:                    strings.TrimSpace(raw.ID),
		ShopID:                strings.TrimSpace(raw.ShopID),
		LogisticID:            logisticID,
		City:                  parseInt(raw.City),
		Platform:              platform,
		DailyPlatformSequence: parseInt(raw.SourceSN),
		OrderNo:               strings.TrimSpace(raw.SourceID),
		OrderTime:             formatUnixTimestamp(raw.OrderTime),
		UserAddress:           firstNonEmptyString(raw.MapAddress, raw.Address),
		RawShopName:           firstNonEmptyString(raw.Extend.ChannelName, raw.Shop.Name),
		ShopAddress:           raw.Shop.Name,
		RawShopAddress:        raw.Shop.Name,
		IsSubscribe:           parseBool(raw.IsSubscribe),
		CompletedAt:           formatUnixTimestamp(raw.FinishedTime),
		Longitude:             parseFloat(raw.Longitude),
		Latitude:              parseFloat(raw.Latitude),
		Status:                firstNonEmptyString(stripHTML(raw.Tips), raw.StatusStr),
		DeliveryDeadline:      formatDeadline(raw.DeliveryTime),
		DistanceKM:            float64(parseInt(raw.DeliveryDistance)) / 1000,
		DistanceIsLinear:      false,
		ActualPaid:            actualPaid,
		ExpectedIncome:        expectedIncome,
		PlatformCommission:    expectedIncome - actualPaid,
		Delivery:              mapDelivery(raw.Delivery),
		Items:                 []domain.OrderItem{},
	}
}

func mapDelivery(raw rawDelivery) *domain.Delivery {
	if strings.TrimSpace(raw.Track) == "" && strings.TrimSpace(raw.DeliveryName) == "" && strings.TrimSpace(asString(raw.LogisticID)) == "" {
		return nil
	}

	return &domain.Delivery{
		LogisticName: firstNonEmptyString(strings.TrimSpace(raw.LogisticName), strings.TrimSpace(asString(raw.LogisticID))),
		Track:        strings.TrimSpace(raw.Track),
		RiderName:    strings.TrimSpace(raw.DeliveryName),
	}
}

func applyDetailToOrder(order *domain.Order, detail rawDetailOrder) {
	if order == nil {
		return
	}

	if value := strings.TrimSpace(asString(detail.ID)); value != "" {
		order.ID = value
	}
	if value := firstNonEmptyString(strings.TrimSpace(asString(detail.Delivery.LogisticID)), strings.TrimSpace(order.LogisticID)); value != "" {
		order.LogisticID = value
	}
	if value := parseOrderTime(detail.OrderTime); value != "" {
		order.OrderTime = value
	}
	if value := firstNonEmptyString(detail.MapAddress, detail.Address); value != "" {
		order.UserAddress = value
	}
	if value := readPreferredShopNameFromDetail(detail); value != "" {
		order.RawShopName = value
	}
	if value := readPreferredShopAddressFromDetail(detail); value != "" {
		order.ShopAddress = value
		order.RawShopAddress = value
	}
	if value := firstNonEmptyString(strings.TrimSpace(asString(detail.ShopID)), strings.TrimSpace(asString(detail.MerchantID))); value != "" {
		order.ShopID = value
	}
	if value := formatUnixTimestamp(detail.FinishedTime); value != "" {
		order.CompletedAt = value
	}
	if value := parseFloat(detail.Longitude); value != 0 {
		order.Longitude = value
	}
	if value := parseFloat(detail.Latitude); value != 0 {
		order.Latitude = value
	}
	if value := maxFloat(0, parseFloat(detail.DeliveryDistance)/1000); value > 0 {
		order.DistanceKM = value
	}
	if value := firstNonEmptyString(stripHTML(detail.Tips), order.Status); value != "" {
		order.Status = value
	}
	if value := formatDeadline(detail.DeliveryTime); value != "" {
		order.DeliveryDeadline = value
	}
	if value := strings.TrimSpace(detail.DeliveryTimeFormat); value != "" {
		order.DeliveryTimeRange = value
	}

	if delivery := mapDetailDelivery(detail.Delivery); delivery != nil {
		order.Delivery = delivery
	}

	actualPaid, expectedIncome, platformCommission, hasAmount := parseDetailAmountFields(order.Platform, detail.Fee)
	if hasAmount {
		order.ActualPaid = actualPaid
		order.ExpectedIncome = expectedIncome
		order.PlatformCommission = platformCommission
	}

	items := mapDetailItems(detail.Goods)
	if len(items) > 0 {
		order.Items = items
	}
}

func mapDetailDelivery(raw rawDelivery) *domain.Delivery {
	logisticName := strings.TrimSpace(raw.LogisticName)
	sendFee := parseInt(raw.SendFee)
	tip := parseInt(raw.Tip)
	premiumFee := parseInt(raw.PremiumFee)
	pickupTime := parseOrderTime(raw.PickupTime)
	track := strings.TrimSpace(raw.Track)
	riderName := strings.TrimSpace(raw.DeliveryName)

	if logisticName == "" && sendFee <= 0 && tip <= 0 && premiumFee <= 0 && pickupTime == "" && track == "" && riderName == "" {
		return nil
	}

	return &domain.Delivery{
		LogisticName:     logisticName,
		SendFee:          sendFee,
		Tip:              tip,
		PremiumFee:       premiumFee,
		TotalDeliveryFee: sendFee + tip + premiumFee,
		PickupTime:       pickupTime,
		Track:            track,
		RiderName:        riderName,
	}
}

func parseDetailAmountFields(platform string, fee rawDetailFee) (actualPaid, expectedIncome, platformCommission int, hasAmount bool) {
	actualPaid = parseInt(fee.UserFee)
	expectedIncome = parseInt(fee.ShopFee)
	platformCommission = applyJDPlatformCommissionFallback(platform, actualPaid, expectedIncome-actualPaid)
	hasAmount = actualPaid > 0 || expectedIncome > 0
	return actualPaid, expectedIncome, platformCommission, hasAmount
}

func mapDetailItems(goods []rawDetailGoods) []domain.OrderItem {
	items := make([]domain.OrderItem, 0, len(goods))
	for _, item := range goods {
		productName := strings.TrimSpace(item.GoodsName)
		productNo := strings.TrimSpace(item.SKUCode)
		quantity := parseInt(item.Number)
		if quantity <= 0 {
			quantity = 1
		}
		if productName == "" && productNo == "" {
			continue
		}
		items = append(items, domain.OrderItem{
			ProductName: productName,
			ProductNo:   productNo,
			Quantity:    quantity,
			Thumb:       strings.TrimSpace(item.Thumb),
		})
	}
	return items
}

func readPreferredShopNameFromDetail(detail rawDetailOrder) string {
	return firstNonEmptyString(strings.TrimSpace(detail.Extend.ChannelName), strings.TrimSpace(detail.Shop.Name))
}

func readPreferredShopAddressFromDetail(detail rawDetailOrder) string {
	return strings.TrimSpace(detail.Shop.Name)
}

func mapRawQueryOrder(raw map[string]any) domain.Order {
	platform := normalizePlatformName(getString(raw, "channel_tag_name"))
	actualPaid, expectedIncome, platformCommission := parseAmountFields(platform, raw)

	return domain.Order{
		ID:                    getString(raw, "id"),
		ShopID:                firstNonEmptyString(getString(raw, "shop_id"), getString(raw, "merchant_id")),
		LogisticID:            getString(raw, "delivery_id"),
		City:                  maxInt(0, parseInt(raw["city"])),
		Platform:              platform,
		DailyPlatformSequence: parseInt(raw["source_sn"]),
		OrderNo:               getString(raw, "source_id"),
		OrderTime:             parseOrderTime(raw["order_time"]),
		UserAddress:           firstNonEmptyString(getString(raw, "map_address"), getString(raw, "address")),
		RawShopName:           readPreferredShopName(raw),
		ShopAddress:           readPreferredShopAddress(raw),
		RawShopAddress:        readPreferredShopAddress(raw),
		IsSubscribe:           parseBool(raw["is_subscribe"]),
		CompletedAt:           readCompletedAt(raw),
		Longitude:             parseFloat(raw["longitude"]),
		Latitude:              parseFloat(raw["latitude"]),
		Status:                stripHTML(getString(raw, "tips")),
		DeliveryDeadline:      formatDeadline(raw["delivery_time"]),
		DeliveryTimeRange:     strings.TrimSpace(getString(raw, "delivery_time_format")),
		DistanceKM:            maxFloat(0, parseFloat(raw["delivery_distance"])/1000),
		DistanceIsLinear:      false,
		ActualPaid:            actualPaid,
		ExpectedIncome:        expectedIncome,
		PlatformCommission:    platformCommission,
		Items:                 []domain.OrderItem{},
	}
}

func normalizePlatformName(platform string) string {
	value := strings.TrimSpace(platform)
	switch value {
	case "":
		return "未知"
	case "淘宝闪购":
		return "淘宝"
	default:
		return value
	}
}

func parseAmountFields(platform string, raw map[string]any) (actualPaid, expectedIncome, platformCommission int) {
	fee, _ := raw["fee"].(map[string]any)
	actualPaid = parseInt(firstNonNil(fee["user_fee"], raw["user_fee"], fee["total_fee"], raw["total_price"]))
	expectedIncome = parseInt(firstNonNil(fee["shop_fee"], raw["shop_fee"], raw["balance_price"]))
	platformCommission = parseInt(firstNonNil(fee["commission"], raw["commission"]))
	if platformCommission == 0 {
		platformCommission = expectedIncome - actualPaid
	}
	platformCommission = applyJDPlatformCommissionFallback(platform, actualPaid, platformCommission)
	return actualPaid, expectedIncome, platformCommission
}

func applyJDPlatformCommissionFallback(platform string, actualPaid int, platformCommission int) int {
	if strings.TrimSpace(platform) != "京东" || platformCommission != 0 || actualPaid <= 0 {
		return platformCommission
	}
	return -int((float64(actualPaid-100) * 0.06) + 100 + 0.5)
}

func parseOrderTime(value any) string {
	text := strings.TrimSpace(asString(value))
	if strings.Contains(text, "-") {
		return text
	}
	return formatUnixTimestamp(value)
}

func readPreferredShopName(raw map[string]any) string {
	if extend, ok := raw["extend"].(map[string]any); ok {
		if value := firstNonEmptyString(getStringFromMap(extend, "channel_name")); value != "" {
			return value
		}
	}
	if channel, ok := raw["channel"].(map[string]any); ok {
		if value := firstNonEmptyString(getStringFromMap(channel, "name")); value != "" {
			return value
		}
	}
	if shop, ok := raw["shop"].(map[string]any); ok {
		if value := firstNonEmptyString(getStringFromMap(shop, "name")); value != "" {
			return value
		}
	}
	return firstNonEmptyString(
		getString(raw, "shop_name"),
		getString(raw, "channel_name"),
		getString(raw, "shopName"),
		getString(raw, "storeName"),
		getString(raw, "merchantName"),
		getString(raw, "merchant_name"),
	)
}

func readPreferredShopAddress(raw map[string]any) string {
	if shop, ok := raw["shop"].(map[string]any); ok {
		if value := firstNonEmptyString(getStringFromMap(shop, "name")); value != "" {
			return value
		}
	}
	return firstNonEmptyString(
		getString(raw, "shop_name"),
		getString(raw, "shopAddress"),
		getString(raw, "storeAddress"),
		getString(raw, "merchantAddress"),
		getString(raw, "shop_address"),
		getString(raw, "store_address"),
		getString(raw, "merchant_address"),
	)
}

func readCompletedAt(raw map[string]any) string {
	if delivery, ok := raw["delivery"].(map[string]any); ok {
		if value := formatUnixTimestamp(delivery["finished_time"]); value != "" {
			return value
		}
	}
	if value := formatUnixTimestamp(raw["finished_time"]); value != "" {
		return value
	}

	value := getString(raw, "finished_time")
	if value == "0" {
		return ""
	}
	return value
}

func stripHTML(value string) string {
	cleaned := htmlTagPattern.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(strings.TrimSpace(cleaned)), " ")
}

func formatUnixTimestamp(value any) string {
	seconds := int64(parseInt(value))
	if seconds <= 0 {
		return ""
	}

	return time.Unix(seconds, 0).Format("2006-01-02 15:04:05")
}

func formatDeadline(value any) string {
	formatted := formatUnixTimestamp(value)
	if formatted == "" {
		return ""
	}
	if len(formatted) < 16 {
		return formatted
	}
	return formatted[5:16]
}

func parseBool(value any) bool {
	text := strings.TrimSpace(asString(value))
	return text == "1" || strings.EqualFold(text, "true")
}

func maxInt(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func maxFloat(a, b float64) float64 {
	if b > a {
		return b
	}
	return a
}

func parseInt(value any) int {
	text := strings.TrimSpace(asString(value))
	if text == "" {
		return 0
	}

	if parsed, err := strconv.Atoi(text); err == nil {
		return parsed
	}

	if floatValue, err := strconv.ParseFloat(text, 64); err == nil {
		return int(floatValue)
	}

	return 0
}

func parseFloat(value any) float64 {
	text := strings.TrimSpace(asString(value))
	if text == "" {
		return 0
	}

	parsed, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}

	return parsed
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func getString(values map[string]any, key string) string {
	return strings.TrimSpace(asString(values[key]))
}

func getStringFromMap(values map[string]any, key string) string {
	return strings.TrimSpace(asString(values[key]))
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value == nil {
			continue
		}
		if strings.TrimSpace(asString(value)) != "" {
			return value
		}
	}
	return nil
}

func extractFirstSubmatch(pattern *regexp.Regexp, text string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func isAllowedStatus(status domain.OrderStatus) bool {
	for _, candidate := range domain.AllowedOrderStatuses() {
		if status == candidate {
			return true
		}
	}
	return false
}
