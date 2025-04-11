package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

type ConvertRequest struct {
	Amount float64 `json:"amount"` // сумма
	From   string  `json:"from"`   // исходная валюта, например: "RUB"
	To     string  `json:"to"`     // целевая валюта, например: "USDT"
}

type ConvertResponse struct {
	ConvertedAmount float64 `json:"convertedAmount"` // результат
}

func main() {
	// Используем переменную окружения PORT или 8080 по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // если переменная окружения PORT не установлена, используем 8080
	}

	r := gin.Default()

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

		// Формируем запрос
		url := "https://api.coingecko.com/api/v3/simple/price?ids=" + currencyID(to) + "&vs_currencies=" + from
		client := resty.New()
		log.Println("Запрос к API CoinGecko:", url) // Логируем запрос

		resp, err := client.R().
			SetHeader("Accept", "application/json").
			SetResult(map[string]map[string]float64{}).
			Get(url)

		if err != nil || resp.IsError() {
			log.Println("Ошибка при получении курса:", err)
			log.Println("Ответ от API:", resp) // Логируем ответ от API
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить курс"})
			return
		}

		data := *resp.Result().(*map[string]map[string]float64)
		rate := data[currencyID(to)][from]
		if rate == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Некорректный курс"})
			return
		}

		converted := req.Amount / rate
		c.JSON(http.StatusOK, ConvertResponse{ConvertedAmount: converted})
	})

	log.Println("🚀 Сервер запущен на http://localhost:" + port)
	r.Run(":" + port) // Запуск на порту, который указан в переменной окружения
}

// currencyID сопоставляет тикер (например, "usdt") с CoinGecko ID
func currencyID(symbol string) string {
	switch strings.ToLower(symbol) {
	case "usdt":
		return "tether"
	case "btc":
		return "bitcoin"
	case "eth":
		return "ethereum"
	default:
		return strings.ToLower(symbol) // по умолчанию
	}
}
