import React from "react";
import "./StormTracker.css";

const StormMap = ({ region, storm }) => {
  // Логи для отладки
  console.log(`Rendering StormMap for region: ${region}`, storm);

  // Проверяем, что storm существует и содержит валидные данные
  const hasData = storm && typeof storm.lat === "number" && storm.lat !== 0 && storm.lon !== 0;

  return (
    <div className="storm-card">
      <h2>{region} Storm</h2>

      {hasData ? (
        <p>
          <strong>Latitude:</strong> {storm.lat} <br />
          <strong>Longitude:</strong> {storm.lon} <br />
          <strong>Temperature:</strong> {storm.temp.toFixed(1)} °C <br />
          <strong>Humidity:</strong> {storm.humidity} % <br />
          <strong>Wind Speed:</strong> {storm.wind_kmh} km/h <br />
          <strong>Timestamp:</strong> {storm.timestamp || "N/A"} <br />
        </p>
      ) : (
        <div>
          <p>No storm data</p>
          <pre>{JSON.stringify(storm, null, 2)}</pre> {/* Показ объекта для отладки */}
        </div>
      )}
    </div>
  );
};

export default StormMap;