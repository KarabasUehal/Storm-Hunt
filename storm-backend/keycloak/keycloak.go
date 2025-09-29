package keycloak

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog/log"
)

// JWKS кэш и синхронизация
var (
	jwksURL       string            // URL для получения ключей
	jwksCache     jwk.Set           // Кэш для хранения ключей
	jwksCacheTime time.Time         // Время обновления кэша для проверки TTL
	jwksMutex     sync.RWMutex      // Mutex для безопасной работы с кэшем
	cacheTTL      = 5 * time.Minute // Кэш на 5 минут
)

// Инициализация JWKS
func InitJWKS() {
	keycloakURL := os.Getenv("KEYCLOAK_URL") // Получение url из переменных окружения
	if keycloakURL == "" {
		log.Fatal().Msg("KEYCLOAK_URL env var not set") // Завершение работы, если url для keycloak не задан
	}
	jwksURL = keycloakURL + "/realms/stormhunter-realm/protocol/openid-connect/certs" // Определяем эндпоинт для получения ключей
	log.Info().Msgf("Initializing JWKS from: %s", jwksURL)

	if err := FetchJWKS(); err != nil { // Загрузка JWKS при старте
		log.Fatal().Err(err).Msg("Failed to fetch JWKS at startup") // Завершение работы при ошибке
	}
}

// Загрузка и кэширование JWKS
func FetchJWKS() error {
	jwksMutex.Lock()         // Блокировка Mutex, чтобы безопасно работать с кэшем
	defer jwksMutex.Unlock() // Разблокировка через оператор отложенных функций

	if jwksCache != nil && time.Since(jwksCacheTime) < cacheTTL { // Проверка кэша на актуальность по TTL
		log.Info().Msg("Using cached JWKS") // Получение данных из кэша, если он актуален
		return nil
	}

	resp, err := http.Get(jwksURL) // Запрос JWKS, если данные в кэше устарели
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err) // Возвращение ошибки, если она есть
	}
	defer resp.Body.Close() // Закрытие тела ответа

	if resp.StatusCode != http.StatusOK { // Проверка статуса ответа
		return fmt.Errorf("JWKS request failed with status: %s", resp.Status) // Возвращение ошибки, если она есть
	}

	set, err := jwk.ParseReader(resp.Body) // Парсинг JWKS
	if err != nil {
		return fmt.Errorf("failed to parse JWKS: %w", err) // Возвращение ошибки, если она есть
	}

	jwksCache = set            // Сохранение ключей в глобальную переменную
	jwksCacheTime = time.Now() // Отмечается момент обновления кэша
	log.Info().Msg("JWKS fetched and cached successfully")
	return nil // Возвращаем отстутсвие ошибки и обновлённые данные jwks
}

// Возвращение RSA Public Key по kid
func FetchRSAPubKeyFromJWKS(kid string) (*rsa.PublicKey, error) {
	jwksMutex.RLock() // Блокировка Mutex на чтение

	if jwksCache == nil || time.Since(jwksCacheTime) >= cacheTTL { // Проверка, не устарел ли кэш
		jwksMutex.RUnlock() // Разблокировка Mutex на чтение для получения ключей
		if err := FetchJWKS(); err != nil {
			return nil, err
		}
		jwksMutex.RLock() // Повторная блокировка Mutex на чтение после получения ключей
	}
	defer jwksMutex.RUnlock() // Гарантированная разблокировка с помощью оператора отложенных функций

	key, ok := jwksCache.LookupKeyID(kid) // Поиск ключа по его идентификатору
	if !ok {
		return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
	}

	if key.KeyType() != jwa.RSA { // Проверка, что ключ — RSA
		return nil, fmt.Errorf("key with kid %s is not RSA, got %s", kid, key.KeyType())
	}

	var rsaKey rsa.PublicKey
	if err := key.Raw(&rsaKey); err != nil { // Конвертиртация jwk.Key в rsa.PublicKey
		return nil, fmt.Errorf("failed to convert key to RSA: %w", err)
	}

	log.Printf("Fetched RSA public key for kid: %s", kid)
	return &rsaKey, nil // Возвращение указателя на rsa.PublicKey и отсутствие ошибки
}
