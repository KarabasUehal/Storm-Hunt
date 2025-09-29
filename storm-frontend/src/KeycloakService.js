import Keycloak from "keycloak-js";

const keycloak = new Keycloak({
  url: "http://localhost:8081",
  realm: "stormhunter-realm",
  clientId: "storm-client",
});

export const getUser = () => {
  return new Promise((resolve) => {
    if (keycloak.authenticated) {
      resolve({
        token: keycloak.token,
        expired: keycloak.isTokenExpired(),
      });
    } else {
      resolve(null);
    }
  });
};

export default keycloak;