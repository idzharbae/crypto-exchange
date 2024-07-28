package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/idzharbae/crypto-exchange/src/internal/entity"
	"github.com/labstack/echo/v4"
	"github.com/palantir/stacktrace"
)

func main() {
	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		c.Logger().Error(err)
	}

	ex := NewExchange()
	e.POST("/order", ex.handlePlaceOrder)

	e.GET("/book/:market", ex.handleGetBook)

	e.DELETE("/order/cancel/:id", ex.handleCancelOrder)

	e.Start(":3000")
}

type Market string

const (
	MarketETH Market = "ETH"
)

type Exchange struct {
	orderBooks map[Market]*entity.OrderBook
}

type PlaceOrderRequest struct {
	Type      entity.OrderType      `json:"type"`
	Placement entity.OrderPlacement `json:"placement"`
	Size      float64               `json:"size"`
	Price     float64               `json:"price"`
	Market    Market                `json:"market"`
}

type OrderData struct {
	ID             int64                 `json:"id"`
	OrderPlacement entity.OrderPlacement `json:"order_placement"`
	Size           float64               `json:"size"`
	Price          float64               `json:"price"`
	Timestamp      int64                 `json:"timestamp"`
}

type OrderBookData struct {
	Asks []*OrderData `json:"asks"`
	Bids []*OrderData `json:"bids"`

	BidTotalVolume float64
	AskTotalVolume float64
}

func NewExchange() *Exchange {
	orderBooks := make(map[Market]*entity.OrderBook)
	orderBooks[MarketETH] = entity.NewOrderBook(string(MarketETH))
	return &Exchange{
		orderBooks: orderBooks,
	}
}

func (ex *Exchange) handleGetBook(c echo.Context) error {
	market := Market(strings.ToUpper(c.Param("market")))
	orderBook, exist := ex.orderBooks[market]
	if !exist {
		return c.JSON(http.StatusNotFound, map[string]any{
			"msg": "market not found",
		})
	}

	orderBookData := OrderBookData{
		Asks:           []*OrderData{},
		Bids:           []*OrderData{},
		BidTotalVolume: orderBook.BidTotalVolume(),
		AskTotalVolume: orderBook.AskTotalVolume(),
	}

	for _, limit := range orderBook.Asks() {
		for _, order := range limit.Orders {
			orderBookData.Asks = append(orderBookData.Asks, &OrderData{
				ID:             order.ID,
				OrderPlacement: order.OrderPlacement,
				Size:           order.Size,
				Price:          limit.Price,
				Timestamp:      order.Timestamp,
			})
		}
	}

	for _, limit := range orderBook.Bids() {
		for _, order := range limit.Orders {
			orderBookData.Bids = append(orderBookData.Bids, &OrderData{
				ID:             order.ID,
				OrderPlacement: order.OrderPlacement,
				Size:           order.Size,
				Price:          limit.Price,
				Timestamp:      order.Timestamp,
			})
		}
	}

	return c.JSON(200, orderBookData)
}

func (ex *Exchange) handlePlaceOrder(c echo.Context) error {
	var placeOrderRequest PlaceOrderRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&placeOrderRequest); err != nil {
		return err
	}

	orderBook := ex.orderBooks[placeOrderRequest.Market]
	if orderBook == nil {
		return errors.New("order book is empty")
	}

	order := entity.NewOrder(placeOrderRequest.Placement, placeOrderRequest.Size)

	if placeOrderRequest.Type == entity.LimitOrder {
		err := orderBook.PlaceLimitOrder(placeOrderRequest.Price, order)
		if err != nil {
			c.JSON(500, map[string]any{
				"msg": "failed to place order",
			})
			return stacktrace.Propagate(err, "handlePlaceOrder: failed to place limit order")
		}

		return c.JSON(200, map[string]any{
			"msg":   "order placed",
			"order": *order,
		})
	} else if placeOrderRequest.Type == entity.MarketOrder {
		matches, err := orderBook.PlaceMarketOrder(order)
		if err != nil {
			c.JSON(500, map[string]any{
				"msg": "failed to place order",
			})
			return stacktrace.Propagate(err, "handlePlaceOrder: failed to place market order")
		}

		return c.JSON(200, map[string]any{
			"msg":     "order placed",
			"order":   *order,
			"matches": len(matches),
		})
	}

	return c.JSON(400, map[string]any{
		"msg": "invalid order type",
	})
}

func (ex *Exchange) handleCancelOrder(c echo.Context) error {
	orderId := c.Param("id")
	if orderId == "" {
		return c.JSON(http.StatusNotFound, map[string]any{
			"msg": "order ID not found",
		})
	}

	orderIdInt64, err := strconv.ParseInt(orderId, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"msg": "invalid order_id",
		})
	}

	order, exists := entity.OrderIndex[orderIdInt64]
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]any{
			"msg": "order id not found",
		})
	}

	orderBook, exists := ex.orderBooks[Market(order.Market)]
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]any{
			"msg": "market not found",
		})
	}

	err = orderBook.CancelOrderByID(orderIdInt64, order.Order.OrderPlacement)
	if err == entity.ErrNotFound {
		return c.JSON(http.StatusNotFound, map[string]any{
			"msg": "order id not found",
		})
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, map[string]any{
			"msg": "error occured when executing order cancelation",
		})
		return stacktrace.Propagate(err, "handleCancelOrder: failed to cancel order id %d", orderIdInt64)
	}

	return c.JSON(200, map[string]any{
		"msg": "order deleted",
	})
}
