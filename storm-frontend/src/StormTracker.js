import React, { useEffect, useState, useRef } from "react";
import { startStream, getStormWebcams, sendWebcamsTask } from "./StormClient";
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
  const [webcams, setWebcams] = useState({ Atlantic: [], Pacific: [] });
  const [selectedCam] = useState(null);
  const imgRef = useRef(null); // Для рефреша изображения

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalImageUrl, setModalImageUrl] = useState(null);

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
  console.log("Updated stormData:", updated); 
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

  const fetchWebcams = async (region, lat = 0, lon = 0, retryCount = 0) => {
    if (!keycloak.authenticated) return;
    try {
      const cams = await getStormWebcams(region, lat, lon, keycloak.token);
      console.log(`Fetched ${cams.length} cams for ${region}`);

      console.log("Full cams response:", cams);
      if (cams.length > 0) {
        console.log("First cam fields:", cams[0]);
      }

      if (cams.length > 0) {
        for (const cam of cams) {
          try {
            await sendWebcamsTask(region, keycloak.tokenParsed?.sub || 'test', cam.id, keycloak.token);
            console.log(`✅ Task sent for ${cam.id}`);
          } catch (e) {
            console.warn(`⚠️ Task skipped for ${cam.id}: ${e.message}`); 
          }
        }
      }

      setWebcams(prev => ({ ...prev, [region]: cams || [] }));

      if (cams.length === 0 && retryCount < 3) {
        console.log(`Empty for ${region}, retry ${retryCount + 1}/3 in 3s`);
        setTimeout(() => fetchWebcams(region, lat, lon, retryCount + 1), 3000);
      }
    } catch (error) {
      console.error(`Webcams fetch error for ${region}:`, error);
      setWebcams(prev => ({ ...prev, [region]: [] }));
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
            startStreamForRegion("Miami");
            startStreamForRegion("Honolulu");
            startStreamForRegion("London");
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

  useEffect(() => {
    if (!selectedCam || !imgRef.current) return;

    const interval = setInterval(() => {
      if (imgRef.current) {
        imgRef.current.src = `${selectedCam}?t=${Date.now()}`;
      }
    }, 30000); // 30 сек

    return () => clearInterval(interval);
  }, [selectedCam]);

  const openModal = (url) => {
    setModalImageUrl(url);
    setIsModalOpen(true);
  };

  const closeModal = () => {
    setIsModalOpen(false);
    setModalImageUrl(null);
  };

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

    <div className="tracker-maps">
      <div className="region-block">
        <h2>Atlantic Region</h2>
        <StormMap region="Miami" storm={stormData["Miami"]} />
        <div className="region-actions">
          <button
            className="tracker-btn fetch"
            onClick={() =>
              fetchWebcams("Miami", stormData["Miami"]?.lat ?? 0, stormData["Miami"]?.lon ?? 0, 0)
            }
          >
            🎥 Load Nearby Webcams
          </button>
        </div>
        <StormMap region="London" storm={stormData["London"]} />
        <div className="region-actions">
          <button
            className="tracker-btn fetch"
            onClick={() =>
              fetchWebcams("London", stormData["London"]?.lat ?? 0, stormData["London"]?.lon ?? 0, 0)
            }
          >
            🎥 Load Nearby Webcams
          </button>
        </div>
      </div>

      <div className="region-block">
        <h2>Pacific Region</h2>
        <StormMap region="Honolulu" storm={stormData["Honolulu"]} />
        <div className="region-actions">
          <button
            className="tracker-btn fetch"
            onClick={() =>
              fetchWebcams("Honolulu", stormData["Honolulu"]?.lat ?? 0, stormData["Honolulu"]?.lon ?? 0, 0)
            }
          >
            🎥 Load Nearby Webcams
          </button>
        </div>
      </div>
    </div>

    <div className="webcam-section">
      <h2>Live Storm Webcams (USGS CoastCams Snapshots)</h2>
      {Object.values(webcams).flat().length > 0 ? (
        <ul className="webcam-list">
          {Object.values(webcams).flat().map((cam) => (
            <li key={cam.id} className="webcam-item">
              <div className="webcam-info">
                <strong>{cam.name}</strong> — {cam.status === "active" ? "🟢 Active Snapshot" : "🔴 Offline"}<br />
                <small>
                  ({cam.lat.toFixed(2)}, {cam.lon.toFixed(2)})
                </small>
              </div>
              {cam.status === "active" && (cam.streamUrl || cam.stream_url) && (
                <button
                  className="tracker-btn play"
                  onClick={() => openModal(cam.streamUrl || cam.stream_url)}
                >
                  📸 View Snapshot
                </button>
              )}
            </li>
          ))}
        </ul>
      ) : (
        <p>No cams loaded yet — retrying... 
          <button 
            className="tracker-btn fetch" 
            onClick={() => {
              // Retry for both regions
              fetchWebcams("Atlantic", stormData["Atlantic"]?.lat ?? 0, stormData["Atlantic"]?.lon ?? 0, 0);
              fetchWebcams("Pacific", stormData["Pacific"]?.lat ?? 0, stormData["Pacific"]?.lon ?? 0, 0);
            }}
          >
            🔄 Refresh All
          </button>
        </p>
      )}
    </div>

    {isModalOpen && modalImageUrl && (
        <div className="modal-overlay" onClick={closeModal}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <button className="modal-close" onClick={closeModal}>×</button>
            <img 
              src={`${modalImageUrl}?t=${Date.now()}`} // Auto-refresh
              alt="Full-size CoastCam Snapshot"
              className="modal-image"
            />
          </div>
        </div>
      )}
    </div>
  );
};

export default StormTracker;