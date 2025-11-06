import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { createPromiseClient } from "@connectrpc/connect";
import { StormService } from "./gen/storm_connect.js";

const transport = createGrpcWebTransport({
  baseUrl: "http://localhost:8080", 
  useBinaryFormat: true,
  credentials: "include",
});

export const client = createPromiseClient(StormService, transport);

export async function startStream(region, userId, token, onData, signal) {
  const headers = { Authorization: `Bearer ${token}` };
  try {
    const stream = client.startStream({ region, user_id: userId }, { headers, signal });
    for await (const msg of stream) {
  console.log("🌪 Received update:", {
    region: msg.region,
    temp: msg.temp,
    humidity: msg.humidity,
    lat: msg.lat,
    lon: msg.lon,
    wind_kmh: msg.wind_kmh ?? msg.windKmh,
    timestamp: msg.timestamp,
  });
  onData(msg);
}
  } catch (err) {
  if (err.message && /cancel/i.test(err.message)) {
    console.log("⏹️ Stream cancelled by user — normal stop");
  } else {
    console.error("🔥 gRPC stream error:", err);
  }
 }
}

export async function getStormWebcams(region, latitude, longitude, token) {
  try {
    const headers = new Headers({
      Authorization: `Bearer ${token}`,
    });

    const response = await client.getStormWebcams(
      { region, latitude, longitude },
      { headers }
    );

    console.log("🎥 gRPC-Webcams response:", response);
    return response.webcams ?? [];
  } catch (err) {
    console.error("gRPC-Webcams Error:", err);
    throw err;
  }
}

export async function sendWebcamsTask(region, userId, cameraID, token) {
  try {
    const response = await fetch("http://localhost:8080/send-webcams-task", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ region, user_id: userId, cameraID }),
    });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to send webcams task: ${response.status} ${text}`);
    }

    const data = await response.json();
    console.log(`✅ Webcams task sent for ${region}, cameraID=${cameraID}:`, data);
    return data;
  } catch (err) {
    console.error(`🔥 Error sending webcams task for ${region}:`, err);
    throw err;
  }
}