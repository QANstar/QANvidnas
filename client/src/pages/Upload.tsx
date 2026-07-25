import { useState, useRef } from 'react';
import { uploadAPI } from '../api/client';
import './Upload.css';

export default function Upload() {
  const [targetDir, setTargetDir] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [message, setMessage] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(e.target.files || []);
    setFiles(selected);
    setMessage('');
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const dropped = Array.from(e.dataTransfer.files);
    setFiles(dropped);
    setMessage('');
  };

  const handleUpload = async () => {
    if (!targetDir) {
      setMessage('请选择目标目录');
      return;
    }
    if (files.length === 0) {
      setMessage('请选择文件');
      return;
    }

    setUploading(true);
    setProgress(0);

    for (let i = 0; i < files.length; i++) {
      try {
        await uploadAPI.upload(files[i], targetDir, (pct) => {
          setProgress(Math.round((i * 100 + pct) / files.length));
        });
      } catch (err) {
        console.error('Upload failed:', err);
        setMessage(`上传 ${files[i].name} 失败`);
        setUploading(false);
        return;
      }
    }

    setUploading(false);
    setProgress(100);
    setMessage(`成功上传 ${files.length} 个文件`);
    setFiles([]);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  return (
    <div className="upload-page">
      <h2>📤 上传文件</h2>

      <div className="upload-form">
        <div className="form-group">
          <label>目标目录</label>
          <input
            className="auth-input"
            type="text"
            placeholder="/media/upload 或其他扫描文件夹内的路径"
            value={targetDir}
            onChange={(e) => setTargetDir(e.target.value)}
          />
        </div>

        <div
          className="drop-zone"
          onDrop={handleDrop}
          onDragOver={(e) => e.preventDefault()}
        >
          <p className="drop-icon">📁</p>
          <p>拖拽文件到此处或点击选择</p>
          <p className="drop-hint">支持 MP4, MKV, AVI, MP3, FLAC 等格式，最大 10GB</p>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            accept="video/*,audio/*"
            onChange={handleFileSelect}
            className="file-input"
          />
        </div>

        {files.length > 0 && (
          <div className="file-list">
            <p className="file-count">已选择 {files.length} 个文件</p>
            {files.slice(0, 5).map((f, i) => (
              <div key={i} className="file-item">
                <span>{f.name}</span>
                <span className="file-size">{(f.size / (1024 * 1024)).toFixed(1)} MB</span>
              </div>
            ))}
            {files.length > 5 && <p>...还有 {files.length - 5} 个文件</p>}
          </div>
        )}

        {uploading && (
          <div className="progress-bar-wrapper">
            <div className="progress-bar">
              <div className="progress-fill" style={{ width: `${progress}%` }} />
            </div>
            <span className="progress-text">{progress}%</span>
          </div>
        )}

        {message && (
          <p className={`upload-message ${message.includes('失败') ? 'error' : ''}`}>
            {message}
          </p>
        )}

        <button
          className="btn-upload"
          onClick={handleUpload}
          disabled={uploading || files.length === 0}
        >
          {uploading ? `上传中 ${progress}%` : '开始上传'}
        </button>
      </div>
    </div>
  );
}
