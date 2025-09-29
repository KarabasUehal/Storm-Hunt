import React from "react";
import "./StormTracker.css";

const StormMap = ({ region, storm }) => {
  return (
    <div className="storm-card">
      <h2>{region} Storm</h2>
      {storm ? (
        <p>
          <strong>Latitude:</strong> {storm.latitude} <br />
          <strong>Longitude:</strong> {storm.longitude} <br />
          <strong>Wind Speed:</strong> {storm.windSpeed} km/h <br />
          <small>{storm.timestamp}</small>
        </p>
      ) : (
        <p>No storm data</p>
      )}
    </div>
  );
};

export default StormMap;