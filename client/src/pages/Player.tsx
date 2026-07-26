import { useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useStore } from '../stores';
import { playlistAPI } from '../api/client';
import './Player.css';

export default function Player() {
  const navigate = useNavigate();
  const { player, setCurrentTime, setDuration, setPlaying, setPlayMode } = useStore();
  const { currentMedia, isPlaying, currentTime, duration, volume, isMuted, playMode, playlistId } = player;

  const handleTimeUpdate = useCallback((e: React.SyntheticEvent<HTMLVideoElement>) => {
    const video = e.target as HTMLVideoElement;
    setCurrentTime(video.currentTime);
    setDuration(video.duration || 0);
  }, [setCurrentTime, setDuration]);

  const handlePlayPause = () => setPlaying(!isPlaying);
  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const time = Number(e.target.value);
    setCurrentTime(time);
    const video = document.querySelector('video');
    if (video) video.currentTime = time;
  };

  const handlePlayMode = async (mode: typeof playMode) => {
    setPlayMode(mode);
    if (playlistId) {
      await playlistAPI.update(playlistId, mode);
    }
  };

  const formatTime = (s: number): string => {
    if (!s || !isFinite(s)) return '0:00';
    const h = Math.floor(s / 3600);
    const m = Math.floor((s % 3600) / 60);
    const sec = Math.floor(s % 60);
    if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
    return `${m}:${String(sec).padStart(2, '0')}`;
  };

  if (!currentMedia) {
    return (
      <div className="player-empty">
        <p className="empty-icon">🎬</p>
        <p>未选择播放内容</p>
        <button className="btn-sm btn-primary" onClick={() => navigate('/')}>浏览媒体</button>
      </div>
    );
  }

  const isVideo = currentMedia.type === 'video';
  const streamUrl = `/api/stream/${isVideo ? 'video' : 'audio'}/${currentMedia.id}`;

  return (
    <div className="player-page">
      <div className="player-container">
        {isVideo ? (
          <div className="video-wrapper">
            <video
              src={streamUrl}
              onTimeUpdate={handleTimeUpdate}
              onPlay={() => setPlaying(true)}
              onPause={() => setPlaying(false)}
              onEnded={() => setPlaying(false)}
              autoPlay
              controls={false}
              style={{ width: '100%', maxHeight: '70vh', background: '#000' }}
            />
          </div>
        ) : (
          <div className="audio-visual">
            <div className="audio-cover">
              {currentMedia.coverPath ? (
                <img src={`/api/stream/cover/${currentMedia.id}`} alt={currentMedia.title} />
              ) : (
                <div className="audio-placeholder">🎵</div>
              )}
            </div>
            <audio
              src={streamUrl}
              onTimeUpdate={(e) => {
                const audio = e.target as HTMLAudioElement;
                setCurrentTime(audio.currentTime);
                setDuration(audio.duration || 0);
              }}
              onPlay={() => setPlaying(true)}
              onPause={() => setPlaying(false)}
              autoPlay
            />
            <h2 className="audio-title">{currentMedia.title}</h2>
          </div>
        )}

        {/* Controls */}
        <div className="player-controls">
          {/* Seek bar */}
          <div className="seek-bar-wrapper">
            <input
              type="range"
              className="seek-bar"
              min={0}
              max={duration || 0}
              value={currentTime}
              onChange={handleSeek}
              step={1}
              style={{
                background: duration > 0
                  ? `linear-gradient(to right, var(--pink) ${(currentTime / duration) * 100}%, var(--border-color) ${(currentTime / duration) * 100}%)`
                  : undefined,
              }}
            />
            <div className="time-display">
              <span>{formatTime(currentTime)}</span>
              <span>{formatTime(duration)}</span>
            </div>
          </div>

          {/* Play controls */}
          <div className="control-buttons">
            <button className="ctrl-btn" onClick={handlePlayPause} title={isPlaying ? '暂停' : '播放'}>
              {isPlaying ? '⏸' : '▶'}
            </button>
            <button className="ctrl-btn ctrl-sm" onClick={() => handlePlayMode('sequential')} title="顺序播放">
              {playMode === 'sequential' ? '🔁' : '➡️'}
            </button>
            <button className="ctrl-btn ctrl-sm" onClick={() => handlePlayMode('loop')} title="循环播放">
              {playMode === 'loop' ? '🔄' : '🔂'}
            </button>
            <button className="ctrl-btn ctrl-sm" onClick={() => handlePlayMode('random')} title="随机播放">
              {playMode === 'random' ? '🎲' : '🔀'}
            </button>
          </div>

          {/* Volume */}
          <div className="volume-control">
            <span>{isMuted ? '🔇' : '🔊'}</span>
            <input type="range" min={0} max={1} step={0.01} value={volume} className="volume-bar" />
          </div>
        </div>
      </div>
    </div>
  );
}
