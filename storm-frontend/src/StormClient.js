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