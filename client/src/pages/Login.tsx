import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { authAPI } from '../api/client';
import { useStore } from '../stores';
import './Auth.css';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [needsSetup, setNeedsSetup] = useState(false);
  const navigate = useNavigate();
  const { setAuth, token } = useStore();

  useEffect(() => {
    if (token) { navigate('/'); return; }
    authAPI.checkSetup()
      .then((res) => {
        if (res.data.setup_required) setNeedsSetup(true);
      })
      .catch(() => {});
  }, [token, navigate]);

  if (needsSetup) {
    return (
      <div className="auth-container">
        <div className="auth-card">
          <h1 className="auth-title">🎬 QANvidnas</h1>
          <p className="auth-message">系统尚未初始化，请先 <Link to="/setup">创建管理员</Link></p>
        </div>
      </div>
    );
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const res = await authAPI.login(username, password);
      const userRes = await fetch('/api/auth/me', {
        headers: { Authorization: `Bearer ${res.data.token}` },
      });
      const user = await userRes.json();
      setAuth(user, res.data.token);
      navigate('/');
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setError(axiosErr.response?.data?.error || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1 className="auth-title">🎬 QANvidnas</h1>
        <p className="auth-subtitle">登录到你的媒体库</p>

        <form onSubmit={handleSubmit} className="auth-form">
          <input
            className="auth-input"
            type="text"
            placeholder="用户名"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
          <input
            className="auth-input"
            type="password"
            placeholder="密码"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          {error && <p className="auth-error">{error}</p>}
          <button className="auth-btn" type="submit" disabled={loading}>
            {loading ? '登录中...' : '登录'}
          </button>
        </form>

        <p className="auth-footer">
          没有账号？<Link to="/register">使用注册码注册</Link>
        </p>
      </div>
    </div>
  );
}
