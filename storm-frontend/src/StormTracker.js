import React, { useEffect, useState, useRef } from "react";
import { startStream, streamStormUpdates } from "./StormClient";
import { useKeycloak } from "@react-keycloak/web";
import StormMap from "./StormMap";
import "./StormTracker.css";

function decodeJwt(token) {
  try {
    const payload = token.split(".")[1];
    const decoded = JSON.parse(atob(payload));
    return decoded;
  } catch (e) {
    console.error("Failed to decode token:", e);
    return null;
  }
}

const StormTracker = () => {
  const { keycloak, initialized } = useKeycloak();
  const [stormData, setStormData] = useState({});
  const streamsRef = useRef({});

  // --- Универсальный запуск стрима ---
  const startStreamForRegion = async (region) => {
    try {
      if (!initialized) {
        console.log("⏳ Keycloak not initialized yet");
        return;
      }

      if (!keycloak.authenticated) {
        console.log("🔐 User not authenticated, redirecting to login...");
        keycloak.login();
        return;
      }

      await keycloak.updateToken(30);
      const token = keycloak.token;

      let userId = keycloak.tokenParsed?.sub;
        if (!userId) {
         const decoded = decodeJwt(token);
            userId = decoded?.sub;
          }

           if (!token || !userId) {
             console.error("❌ Token or user ID is missing. Cannot start stream.");
          console.log("Decoded token:", decodeJwt(token)); // для отладки
           return;
           }

      if (!token || !userId) {
        console.error("❌ Token or user ID is missing. Cannot start stream.");
        return;
      }

      // Останавливаем предыдущий поток (если есть)
      stopStream(region);

      // Создаём AbortController для управления потоком
      const controller = new AbortController();
      streamsRef.current[region] = controller;

      console.log(`🚀 Sending startStream for region: ${region}, user: ${userId}`);

      // --- gRPC поток ---
      startStream(
        region,
        userId,
        token,
        (response) => {
          console.log(`🌪 Received response for ${region}:`, response);

          // Нормализация объекта
          const normalizedWeather = {
            region: response.region ?? region,
            temp: response.temp ?? 0,
            humidity: response.humidity ?? 0,
            lat: response.lat ?? 0,
            lon: response.lon ?? 0,
            wind_kmh: response.wind_kmh ?? response.windKmh ?? 0,
            timestamp: response.timestamp ?? "",
          };

          setStormData((prev) => {
            const updated = {
    ...prev,
    [region]: { ...normalizedWeather },
  };
  console.log("Updated stormData:", updated); // Для отладки
  return updated;
          });
        },
        controller.signal
      );
    } catch (err) {
      console.error(`🔥 Stream error for ${region}:`, err);
    }
  };

  // --- Остановка стрима для конкретного региона ---
  const stopStream = (regionToStop) => {
    const controller = streamsRef.current[regionToStop];
    if (controller) {
      controller.abort();
      delete streamsRef.current[regionToStop];
      setStormData((prev) => {
        const updated = { ...prev };
        delete updated[regionToStop];
        return updated;
      });
      console.log(`🛑 Stopped stream for region: ${regionToStop}`);
    }
  };

  // --- useEffect при монтировании ---
  useEffect(() => {
    console.log("useEffect triggered, Keycloak status:", {
      initialized,
      authenticated: keycloak.authenticated,
      token: !!keycloak.token,
    });

    const initStreams = async () => {
      if (initialized && keycloak.authenticated) {
        try {
          await keycloak.updateToken(30);
          if (keycloak.token) {
            console.log("🔑 Token ready, starting storm streams...");
            startStreamForRegion("Atlantic");
            startStreamForRegion("Pacific");
          } else {
            console.error("Failed to refresh token — redirecting to login");
            keycloak.login();
          }
        } catch (err) {
          console.error("Token refresh error:", err);
          keycloak.login();
        }
      } else if (initialized && !keycloak.authenticated) {
        console.log("🔸 Not authenticated, redirecting to login...");
        keycloak.login();
      }
    };

    initStreams();
    // cleanup при размонтировании
    return () => {
      Object.keys(streamsRef.current).forEach((region) => stopStream(region));
    };
  }, [initialized, keycloak]);

  return (
    <div className="tracker-container">
      <h1 className="tracker-title">Storm Tracker 🌪️</h1>

      <div className="tracker-controls">
        <button
          className="tracker-btn start"
          onClick={() => startStreamForRegion("Atlantic")}
        >
          Start Atlantic
        </button>
        <button
          className="tracker-btn stop"
          onClick={() => stopStream("Atlantic")}
        >
          Stop Atlantic
        </button>

        <button
          className="tracker-btn start"
          onClick={() => startStreamForRegion("Pacific")}
        >
          Start Pacific
        </button>
        <button
          className="tracker-btn stop"
          onClick={() => stopStream("Pacific")}
        >
          Stop Pacific
        </button>
      </div>

      <div style={{ display: "flex", flexDirection: "row" }}>
        <StormMap region="Atlantic" storm={stormData["Atlantic"]} />
        <StormMap region="Pacific" storm={stormData["Pacific"]} />
      </div>
    </div>
  );
};

export default StormTracker;