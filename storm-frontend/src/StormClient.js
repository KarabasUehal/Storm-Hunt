import { createPromiseClient } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { StormService } from "./gen/storm_connect.ts";

const transport = createGrpcWebTransport({
  baseUrl: "http://localhost:8080",
  credentials: "include",
});

const client = createPromiseClient(StormService, transport);
/**
 * Запускает поток штормов по заданному региону
 *
 * @param {string} region - Название региона 
 * @param {string} token - JWT токен из Keycloak
 * @param {(data: object) => void} callback - Функция, вызываемая при получении каждого сообщения
 * @param {AbortSignal} [signal] - AbortSignal для остановки стрима
 */

export async function streamStormUpdates(region, token, callback, signal) {
  try {
    if (!token) {
      console.error("No token provided for gRPC-Web request");
      throw new Error("Authentication token is missing");
    }

    console.log("Starting gRPC-Web stream with region:", region, "token:", token);

    const headers = new Headers({
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/grpc-web+proto",
      "x-grpc-web": "1",
    });

    const stream = client.streamStormUpdates({ region }, { headers, signal });

    for await (const response of stream) {
      if (signal?.aborted) {
        console.log(`Stream for ${region} aborted`);
        break;
      }
      console.log("Received stream response:", response);
      callback(response);
    }
  } catch (error) {
    if (signal?.aborted) {
      console.log(`Stream for ${region} stopped by user`);
      return;
    }
    console.error("gRPC Stream Error:", error);
    throw error;
  }
}