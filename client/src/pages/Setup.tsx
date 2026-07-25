import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authAPI } from '../api/client';
import { useStore } from '../stores';
import './Auth.css';

export default function Setup() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [needsSetup, setNeedsSetup] = useState(true);
  const navigate = useNavigate();
  const { setAuth, token } = useStore();

  useEffect(() => {
    if (token) { navigate('/'); return; }
    authAPI.checkSetup()
      .then((res) => setNeedsSetup(res.data.setup_required))
      .catch(() => navigate('/login'));
  }, [token, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError('两次密码不一致');
      return;
    }

    if (password.length < 6) {
      setError('密码至少 6 位');
      return;
    }

    setLoading(true);
    try {
      const res = await authAPI.setup(username, password);
      setAuth({ id: 0, username, is_admin: true }, res.data.token);
      navigate('/');
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      setError(axiosErr.response?.data?.error || '初始化失败');
    } finally {
      setLoading(false);
    }
  };

  if (!needsSetup) {
    return <div className="auth-container"><p className="auth-message">系统已初始化，<a href="/login">去登录</a></p></div>;
  }

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1 className="auth-title">🎬 QANvidnas</h1>
        <p className="auth-subtitle">首次使用，创建管理员账号</p>

        <form onSubmit={handleSubmit} className="auth-form">
          <input
            className="auth-input"
            type="text"
            placeholder="用户名（至少 3 位）"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
            minLength={3}
            maxLength={32}
          />
          <input
            className="auth-input"
            type="password"
            placeholder="密码（至少 6 位）"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={6}
          />
          <input
            className="auth-input"
            type="password"
            placeholder="确认密码"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            required
          />
          {error && <p className="auth-error">{error}</p>}
          <button className="auth-btn" type="submit" disabled={loading}>
            {loading ? '创建中...' : '创建管理员'}
          </button>
        </form>
      </div>
    </div>
  );
}
