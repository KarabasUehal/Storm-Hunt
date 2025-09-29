import React, { useEffect, useState, useRef } from "react";
import { streamStormUpdates } from "./StormClient";
import { useKeycloak } from "@react-keycloak/web";
import StormMap from "./StormMap";
import "./StormTracker.css";

const StormTracker = () => {
  const { keycloak, initialized } = useKeycloak();
  const [stormData, setStormData] = useState({});
  const streamsRef = useRef({});

  const startStream = async (region) => {
    if (!initialized) return;
    if (!keycloak.authenticated) {
      keycloak.login();
      return;
    }
    try {
      if (keycloak.isTokenExpired()) {
        await keycloak.updateToken(30);
      }

      const controller = new AbortController();
      streamsRef.current[region] = controller;

      await streamStormUpdates(
        region,
        keycloak.token,
        (response) => {
          setStormData((prev) => ({
            ...prev,
            [region]: { ...response, region },
          }));
        },
        controller.signal
      );
    } catch (error) {
      console.error("Stream Error:", error);
    }
  };

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
      console.log(`Stopped stream for region: ${regionToStop}`);
    }
  };

  useEffect(() => {
    if (initialized && keycloak.authenticated) {
      startStream("Atlantic");
      startStream("Pacific");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialized, keycloak.authenticated]);

  return (
    <div className="tracker-container">
      <h1 className="tracker-title">Storm Tracker 🌪️</h1>

      <div className="tracker-controls">
        <button className="tracker-btn start" onClick={() => startStream("Atlantic")}>Start Atlantic</button>
        <button className="tracker-btn stop" onClick={() => stopStream("Atlantic")}>Stop Atlantic</button>
        <button className="tracker-btn start" onClick={() => startStream("Pacific")}>Start Pacific</button>
        <button className="tracker-btn stop" onClick={() => stopStream("Pacific")}>Stop Pacific</button>
      </div>

      <div style={{ display: "flex", flexDirection: "row" }}>
        <StormMap region="Atlantic" storm={stormData["Atlantic"]} />
        <StormMap region="Pacific" storm={stormData["Pacific"]} />
      </div>
    </div>
  );
};

export default StormTracker;