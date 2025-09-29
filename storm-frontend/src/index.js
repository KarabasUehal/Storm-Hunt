import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { ReactKeycloakProvider } from '@react-keycloak/web';
import keycloak from './KeycloakService';

const root = ReactDOM.createRoot(document.getElementById('root'));

root.render(
  <ReactKeycloakProvider
    authClient={keycloak}
    initOptions={{
      onLoad: 'login-required',
      pkceMethod: 'S256',
      redirectUri: window.location.origin,
    }}
  >
    <App />
  </ReactKeycloakProvider>
);