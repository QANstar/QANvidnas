import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useStore } from '../stores';
import './Layout.css';

export default function Layout() {
  const { user, logout, filter, setTypeFilter, isTVMode } = useStore();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className={`app-layout ${isTVMode ? 'tv-mode' : ''}`}>
      <nav className="sidebar">
        <div className="logo">
          <span className="logo-icon">🎬</span>
          <span className="logo-text">QANvidnas</span>
        </div>

        <div className="nav-links">
          <NavLink to="/" end className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <span className="nav-icon">🏠</span>
            <span>首页</span>
          </NavLink>
          <NavLink to="/search" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <span className="nav-icon">🔍</span>
            <span>搜索</span>
          </NavLink>
          <NavLink to="/playlists" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <span className="nav-icon">📋</span>
            <span>播放列表</span>
          </NavLink>
          <NavLink to="/upload" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
            <span className="nav-icon">📤</span>
            <span>上传</span>
          </NavLink>
          {user?.is_admin && (
            <NavLink to="/settings" className={({ isActive }) => isActive ? 'nav-item active' : 'nav-item'}>
              <span className="nav-icon">⚙️</span>
              <span>设置</span>
            </NavLink>
          )}
        </div>

        <div className="nav-footer">
          <div className="user-info">
            <span>{user?.username}</span>
            {user?.is_admin && <span className="admin-badge">管理员</span>}
          </div>
          <button onClick={handleLogout} className="btn-logout">退出</button>
        </div>
      </nav>

      <main className="main-content">
        {/* Type filter bar */}
        <div className="filter-bar">
          <button
            className={`filter-btn ${filter.typeFilter === '' ? 'active' : ''}`}
            onClick={() => setTypeFilter('')}
          >
            全部
          </button>
          <button
            className={`filter-btn ${filter.typeFilter === 'video' ? 'active' : ''}`}
            onClick={() => setTypeFilter('video')}
          >
            🎬 视频
          </button>
          <button
            className={`filter-btn ${filter.typeFilter === 'audio' ? 'active' : ''}`}
            onClick={() => setTypeFilter('audio')}
          >
            🎵 音频
          </button>
        </div>

        <Outlet />
      </main>
    </div>
  );
}
