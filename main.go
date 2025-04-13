package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	cache "server_sbpk/chache"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type ConvertRequest struct {
	Amount float64 `json:"amount"` // сумма
	From   string  `json:"from"`   // исходная валюта, например: "RUB"
	To     string  `json:"to"`     // целевая валюта, например: "USDT"
}

type ConvertResponse struct {
	ConvertedAmount float64 `json:"convertedAmount"`
	Currency        string  `json:"currency"`
	Wallet          string  `json:"wallet,omitempty"`
	Message         string  `json:"message"`
}

const OWNER_WALLET = "0x5F6bE5797EDE88B6D9b4aF6cB8e3A9E2b070ac93"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// Разрешаем CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://splendid-peony-e3b7a2.netlify.app", "http://localhost:5173", "http://172.20.10.2:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	}))

	r.POST("/convert", func(c *gin.Context) {
		var req ConvertRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
			return
		}

		from := strings.ToLower(req.From)
		to := strings.ToLower(req.To)

		if from == "" || to == "" || req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Неверные параметры запроса"})
			return
		}

		key := currencyID(to) + "_" + from

		// 🔎 Попробуем получить курс из кэша
		if rate, found := cache.GetCachedRate(key); found {
			converted := req.Amount / rate
			response := ConvertResponse{
				ConvertedAmount: converted,
				Currency:        strings.ToUpper(req.To),
				Message:         fmt.Sprintf("Переведите %.2f на адрес  %s", converted, OWNER_WALLET),
				Wallet:          OWNER_WALLET,
			}
			c.JSON(http.StatusOK, response)
			return
		}

		// 🔄 Если в кэше нет — запрос к CoinGecko
		url := "https://api.coingecko.com/api/v3/simple/price?ids=" + currencyID(to) + "&vs_currencies=" + from
		client := resty.New()

		log.Println("🌐 Запрос к API CoinGecko:", url)

		resp, err := client.R().
			SetHeader("x-cg-pro-api-key", "CG-wmi7LpR5B84uad7kPFE1knYa").
			SetHeader("Accept", "application/json").
			SetResult(map[string]map[string]float64{}).
			Get(url)

		if err != nil || resp.IsError() {
			log.Println("❌ Ошибка при получении курса:", err)
			log.Println("Ответ от API:", resp)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить курс"})
			return
		}

		data := *resp.Result().(*map[string]map[string]float64)
		rate := data[currencyID(to)][from]

		if rate == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Некорректный курс"})
			return
		}

		// 💾 Сохраняем в кэш
		cache.SetCachedRate(key, rate)

		converted := req.Amount / rate

		response := ConvertResponse{
			ConvertedAmount: converted,
			Currency:        strings.ToUpper(req.To),
			Message:         fmt.Sprintf("Переведите %.2f %s на адрес ", converted, OWNER_WALLET),
			Wallet:          OWNER_WALLET,
		}

		c.JSON(http.StatusOK, response)
	})

	log.Println("🚀 Сервер запущен на http://localhost:" + port)
	r.Run(":" + port)
}

// currencyID сопоставляет тикер с CoinGecko ID
func currencyID(symbol string) string {
	switch strings.ToLower(symbol) {
	case "usdt":
		return "tether"
	case "btc":
		return "bitcoin"
	case "eth":
		return "ethereum"
	default:
		return strings.ToLower(symbol)
	}
}
