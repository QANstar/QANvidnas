import { useState, useEffect } from 'react';
import { adminAPI, mediaAPI } from '../api/client';
import './Settings.css';

export default function Settings() {
  const [scanFolders, setScanFolders] = useState<{ id: number; path: string; status: string }[]>([]);
  const [inviteCodes, setInviteCodes] = useState<{ code: string; description: string; maxUses: number; used: number }[]>([]);
  const [newPath, setNewPath] = useState('');
  const [newCode, setNewCode] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [scanning, setScanning] = useState(false);
  const [oldPw, setOldPw] = useState('');
  const [newPw, setNewPw] = useState('');

  const fetchData = async () => {
    try {
      const [foldersRes, codesRes] = await Promise.all([
        adminAPI.getScanFolders(),
        adminAPI.getInviteCodes(),
      ]);
      setScanFolders(foldersRes.data.items);
      setInviteCodes(codesRes.data.items);
    } catch (err) {
      console.error('Failed to load settings:', err);
    }
  };

  useEffect(() => { fetchData(); }, []);

  const addFolder = async () => {
    if (!newPath) return;
    try {
      await adminAPI.addScanFolder(newPath);
      setNewPath('');
      fetchData();
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      alert(axiosErr.response?.data?.error || '添加失败');
    }
  };

  const deleteFolder = async (id: number) => {
    await adminAPI.deleteScanFolder(id);
    fetchData();
  };

  const triggerScan = async () => {
    setScanning(true);
    await mediaAPI.scan();
    setTimeout(() => { setScanning(false); fetchData(); }, 3000);
  };

  const addInviteCode = async () => {
    if (!newCode) return;
    try {
      await adminAPI.createInviteCode({ code: newCode, description: newDesc, maxUses: 5 });
      setNewCode('');
      setNewDesc('');
      fetchData();
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } } };
      alert(axiosErr.response?.data?.error || '添加失败');
    }
  };

  const deleteCode = async (code: string) => {
    await adminAPI.deleteInviteCode(code);
    fetchData();
  };

  const changePassword = async () => {
    if (!oldPw || !newPw) return;
    try {
      // await authAPI.changePassword(oldPw, newPw);
      alert('密码修改功能开发中');
    } catch {
      alert('修改失败');
    }
  };

  return (
    <div className="settings-page">
      <h2>⚙️ 管理设置</h2>

      {/* Scan Folders */}
      <section className="settings-section">
        <h3>📁 扫描文件夹</h3>
        <div className="add-row">
          <input
            className="auth-input"
            placeholder="文件夹路径，如 /media/videos"
            value={newPath}
            onChange={(e) => setNewPath(e.target.value)}
          />
          <button className="btn-sm btn-primary" onClick={addFolder}>添加</button>
        </div>
        {scanFolders.map((f) => (
          <div key={f.id} className="setting-item">
            <span className="setting-path">{f.path}</span>
            <span className="setting-status">{f.status}</span>
            <button className="btn-sm btn-secondary" onClick={() => deleteFolder(f.id)}>删除</button>
          </div>
        ))}
        <button className="btn-sm btn-primary" onClick={triggerScan} disabled={scanning}>
          {scanning ? '扫描中...' : '🔄 手动扫描'}
        </button>
      </section>

      {/* Invite Codes */}
      <section className="settings-section">
        <h3>🔑 注册码管理</h3>
        <div className="add-row">
          <input
            className="auth-input"
            placeholder="注册码"
            value={newCode}
            onChange={(e) => setNewCode(e.target.value)}
          />
          <input
            className="auth-input"
            placeholder="描述（可选）"
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
          />
          <button className="btn-sm btn-primary" onClick={addInviteCode}>生成</button>
        </div>
        {inviteCodes.map((c) => (
          <div key={c.code} className="setting-item">
            <div>
              <code className="invite-code">{c.code}</code>
              {c.description && <span className="code-desc">{c.description}</span>}
            </div>
            <span className="code-usage">{c.used}/{c.maxUses || '∞'}</span>
            <button className="btn-sm btn-secondary" onClick={() => deleteCode(c.code)}>删除</button>
          </div>
        ))}
      </section>

      {/* Change Password */}
      <section className="settings-section">
        <h3>🔒 修改密码</h3>
        <div className="add-row">
          <input
            className="auth-input"
            type="password"
            placeholder="旧密码"
            value={oldPw}
            onChange={(e) => setOldPw(e.target.value)}
          />
          <input
            className="auth-input"
            type="password"
            placeholder="新密码"
            value={newPw}
            onChange={(e) => setNewPw(e.target.value)}
          />
          <button className="btn-sm btn-primary" onClick={changePassword}>修改</button>
        </div>
      </section>
    </div>
  );
}
