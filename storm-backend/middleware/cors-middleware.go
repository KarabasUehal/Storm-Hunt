package middleware

import "net/http"

// Промежуточный обработчик для CORS
func CorsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // Возвращение нового обработчика
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")                                        // Разрешённый источник для запросов
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")                                          // Разрешённые HTTP-методы для кросс-доменных запросов
		w.Header().Set("Access-Control-Allow-Credentials", "true")                                                    // Разрешение отправки учётных данных
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Grpc-Web, x-grpc-web, Accept") // Разрешённые заголовки для запросов

		if r.Method == "OPTIONS" { // Проверка, является ли запрос предварительным (preflight) запросом с методом OPTIONS
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r) // Если запрос не является OPTIONS, то передаётся следующему обработчику в цепочке
	})
}
