import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useStore } from './stores';
import { authAPI } from './api/client';

import Setup from './pages/Setup';
import Login from './pages/Login';
import Register from './pages/Register';
import Browse from './pages/Browse';
import MediaDetail from './pages/MediaDetail';
import Player from './pages/Player';
import Playlists from './pages/Playlists';
import Search from './pages/Search';
import Upload from './pages/Upload';
import Settings from './pages/Settings';
import TVAuth from './pages/TVAuth';

import Layout from './components/Layout';
import ProtectedRoute from './components/ProtectedRoute';

function App() {
  const { token, user, setAuth, logout } = useStore();

  useEffect(() => {
    if (token && !user) {
      authAPI.me()
        .then((res) => setAuth(res.data, token))
        .catch(() => logout());
    }
  }, [token, user, setAuth, logout]);

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/setup" element={<Setup />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="/tv-auth" element={<TVAuth />} />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Layout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Browse />} />
          <Route path="media/:id" element={<MediaDetail />} />
          <Route path="player" element={<Player />} />
          <Route path="playlists" element={<Playlists />} />
          <Route path="search" element={<Search />} />
          <Route path="upload" element={<Upload />} />
          <Route path="settings" element={<Settings />} />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
