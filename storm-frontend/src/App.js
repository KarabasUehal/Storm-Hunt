import React from 'react';
import { useKeycloak } from '@react-keycloak/web';
import StormTracker from './StormTracker';
import "./App.css";

const LogoutButton = () => {
  const { keycloak } = useKeycloak();

  return (
    <button
      className="logout-btn" onClick={() => keycloak.logout()}
    >
      Logout
    </button>
  );
};

const BackgroundVideo = () => (
  <video
    autoPlay
    loop
    muted
    playsInline
    style={{
      position: 'fixed',
      right: 0,
      bottom: 0,
      minWidth: '100%',
      minHeight: '100%',
      zIndex: -1,
      objectFit: 'cover',
      filter: 'brightness(0.6)', // затемнение, чтобы текст был читаемым
    }}
  >
    <source src="/storm-video.mp4" type="video/mp4" />
    Your browser does not support the video tag.
  </video>
);

function App() {
  const { keycloak, initialized } = useKeycloak();

  if (!initialized) {
    return <div style={{ padding: '20px', textAlign: 'center', color: 'white' }}>Loading Keycloak...</div>;
  }

  if (!keycloak.authenticated) {
    return <div style={{ padding: '20px', textAlign: 'center', color: 'white' }}>Redirecting to login...</div>;
  }

  return (
    <>
      <BackgroundVideo />
      <div style={{ position: 'relative', zIndex: 1 }}>
        <LogoutButton />
        <StormTracker />
      </div>
    </>
  );
}

export default App;