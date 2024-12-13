package main

import "multichannel/cmd/typedefs"

// Callback for /stocks
func stocksCallback(req typedefs.Request) interface{} {
	// Simulate a database query to retrieve stock data
	stockData := []map[string]interface{}{
		{"symbol": "AAPL", "price": 150.0},
		{"symbol": "GOOG", "price": 2500.0},
		{"symbol": "AMZN", "price": 3000.0},
	}

	return stockData
}

// Callback for /weather
func weatherCallback(req typedefs.Request) interface{} {
	// Simulate a weather API call to retrieve current weather conditions
	weatherData := map[string]interface{}{
		"temperature": 75.0,
		"humidity":    60.0,
		"conditions":  "Sunny",
	}

	return weatherData
}

// Callback for /crypto
func cryptoCallback(req typedefs.Request) interface{} {
	// Simulate a cryptocurrency API call to retrieve current prices
	cryptoData := []map[string]interface{}{
		{"symbol": "BTC", "price": 50000.0},
		{"symbol": "ETH", "price": 4000.0},
		{"symbol": "LTC", "price": 200.0},
	}

	return cryptoData
}
