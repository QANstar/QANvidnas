import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { authAPI } from '../api/client';
import { useStore } from '../stores';
import './Auth.css';

export default function TVAuth() {
  const [deviceCode, setDeviceCode] = useState('');
  const [status, setStatus] = useState<'input' | 'waiting' | 'authorized' | 'error'>('input');
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const { setAuth, token } = useStore();

  if (token) { navigate('/'); return null; }

  const handleAuthorize = async () => {
    if (!deviceCode.trim()) return;
    try {
      await authAPI.authorizeDeviceCode(deviceCode.trim().toUpperCase());
      setStatus('authorized');
    } catch {
      setError('授权失败，请检查设备码');
    }
  };

  // Poll for authorization
  useEffect(() => {
    if (status !== 'waiting') return;
    const interval = setInterval(async () => {
      try {
        const res = await authAPI.pollDeviceCode(deviceCode);
        if (res.data.status === 'authorized') {
          setStatus('authorized');
          const userRes = await fetch('/api/auth/me', {
            headers: { Authorization: `Bearer ${res.data.token}` },
          });
          const user = await userRes.json();
          setAuth(user, res.data.token);
          navigate('/');
        }
      } catch {
        setStatus('error');
        setError('设备码已过期');
      }
    }, 3000);
    return () => clearInterval(interval);
  }, [status, deviceCode, setAuth, navigate]);

  return (
    <div className="auth-container">
      <div className="auth-card">
        <h1 className="auth-title">📺 TV 登录</h1>

        {status === 'input' && (
          <div>
            <p className="auth-subtitle">在电视上查看到 4 位设备码后，在此输入以授权</p>
            <input
              className="auth-input"
              type="text"
              placeholder="输入 4 位设备码"
              value={deviceCode}
              onChange={(e) => setDeviceCode(e.target.value.toUpperCase())}
              maxLength={4}
              style={{ textAlign: 'center', fontSize: '1.5rem', letterSpacing: '8px' }}
            />
            <button className="auth-btn" onClick={handleAuthorize} style={{ marginTop: 'var(--space-md)' }}>
              确认授权
            </button>
          </div>
        )}

        {status === 'waiting' && (
          <div>
            <p className="auth-subtitle">设备码已生成</p>
            <div className="device-code-display" style={{
              fontSize: '2.5rem',
              letterSpacing: '12px',
              fontFamily: 'monospace',
              color: 'var(--blue-light)',
              margin: 'var(--space-lg) 0',
            }}>
              {deviceCode}
            </div>
            <p className="auth-subtitle">请在 10 分钟内输入此码完成授权</p>
            <p className="auth-message">等待授权中...</p>
          </div>
        )}

        {status === 'authorized' && (
          <p className="auth-message">授权成功！电视正在跳转...</p>
        )}

        {error && <p className="auth-error">{error}</p>}
      </div>
    </div>
  );
}
