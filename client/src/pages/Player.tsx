import { useCallback, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useStore } from '../stores';
import { playlistAPI } from '../api/client';
import './Player.css';

export default function Player() {
  const navigate = useNavigate();
  const {
    player,
    setCurrentTime,
    setDuration,
    setPlaying,
    setVolume,
    setMuted,
    setPlayMode,
    setCurrentMedia,
    playNext,
    playPrevious,
  } = useStore();
  const {
    currentMedia, isPlaying, currentTime, duration, volume, isMuted,
    playMode, playlistId, playlistItems, currentIndex,
  } = player;

  // Refs for direct DOM control of video/audio elements
  const videoRef = useRef<HTMLVideoElement>(null);
  const audioRef = useRef<HTMLAudioElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  // Sync video/audio volume with store state
  useEffect(() => {
    const media = videoRef.current || audioRef.current;
    if (media) {
      media.volume = volume;
      media.muted = isMuted;
    }
  }, [volume, isMuted]);

  // Sync play/pause with the DOM element
  useEffect(() => {
    const media = videoRef.current || audioRef.current;
    if (!media) return;
    if (isPlaying && media.paused) {
      media.play().catch(() => {});
    } else if (!isPlaying && !media.paused) {
      media.pause();
    }
  }, [isPlaying, currentMedia?.id]);

  // When media changes, load the new source (handled by src change + autoPlay)
  useEffect(() => {
    const media = videoRef.current || audioRef.current;
    if (media && currentMedia) {
      media.load();
      media.play().catch(() => {});
    }
  }, [currentMedia?.id]);

  const handleTimeUpdate = useCallback((e: React.SyntheticEvent<HTMLVideoElement | HTMLAudioElement>) => {
    const media = e.target as HTMLMediaElement;
    setCurrentTime(media.currentTime);
    setDuration(media.duration || 0);
  }, [setCurrentTime, setDuration]);

  const handlePlayPause = () => {
    const media = videoRef.current || audioRef.current;
    if (!media) return;
    if (isPlaying) {
      media.pause();
      setPlaying(false);
    } else {
      media.play().catch(() => {});
      setPlaying(true);
    }
  };

  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const time = Number(e.target.value);
    setCurrentTime(time);
    const media = videoRef.current || audioRef.current;
    if (media) media.currentTime = time;
  };

  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const v = Number(e.target.value);
    setVolume(v);
    const media = videoRef.current || audioRef.current;
    if (media) {
      media.volume = v;
      if (v > 0 && isMuted) {
        media.muted = false;
        setMuted(false);
      }
    }
  };

  const handleMuteToggle = () => {
    const media = videoRef.current || audioRef.current;
    if (!media) return;
    const newMuted = !isMuted;
    media.muted = newMuted;
    setMuted(newMuted);
  };

  const handleFullscreen = () => {
    if (!containerRef.current) return;
    if (document.fullscreenElement) {
      document.exitFullscreen().catch(() => {});
    } else {
      containerRef.current.requestFullscreen().catch(() => {});
    }
  };

  const handlePlayMode = async (mode: typeof playMode) => {
    setPlayMode(mode);
    if (playlistId) {
      await playlistAPI.update(playlistId, mode);
    }
  };

  const handlePrevious = () => {
    const media = playPrevious();
    if (!media) {
      // If no previous, restart current
      const el = videoRef.current || audioRef.current;
      if (el) {
        el.currentTime = 0;
        el.play().catch(() => {});
        setPlaying(true);
      }
    }
  };

  const handleNext = () => {
    playNext();
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
  const token = localStorage.getItem('token') || '';
  const streamUrl = `/api/stream/${isVideo ? 'video' : 'audio'}/${currentMedia.id}?token=${token}`;
  const hasPlaylist = playlistItems.length > 1;

  return (
    <div className="player-page">
      <div className="player-container" ref={containerRef}>
        {isVideo ? (
          <div className="video-wrapper">
            <video
              ref={videoRef}
              src={streamUrl}
              onTimeUpdate={handleTimeUpdate}
              onPlay={() => setPlaying(true)}
              onPause={() => setPlaying(false)}
              onEnded={() => {
                setPlaying(false);
                // Auto-play next if in a playlist
                if (hasPlaylist && playMode !== 'single-loop') {
                  playNext();
                }
              }}
              autoPlay
              controls={false}
              style={{ width: '100%', maxHeight: '70vh', background: '#000' }}
            />
          </div>
        ) : (
          <div className="audio-visual">
            <div className="audio-cover">
              {currentMedia.coverPath ? (
                <img src={`/api/stream/cover/${currentMedia.id}?token=${token}`} alt={currentMedia.title} />
              ) : (
                <div className="audio-placeholder">🎵</div>
              )}
            </div>
            <audio
              ref={audioRef}
              src={streamUrl}
              onTimeUpdate={handleTimeUpdate}
              onPlay={() => setPlaying(true)}
              onPause={() => setPlaying(false)}
              onEnded={() => {
                setPlaying(false);
                if (hasPlaylist && playMode !== 'single-loop') {
                  playNext();
                }
              }}
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
                  ? `linear-gradient(to right, var(--blue) ${(currentTime / duration) * 100}%, var(--border-color) ${(currentTime / duration) * 100}%)`
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
            {/* Previous */}
            <button
              className="ctrl-btn ctrl-sm"
              onClick={handlePrevious}
              disabled={!hasPlaylist && currentTime < 3}
              title={hasPlaylist ? '上一集' : '重新播放'}
            >
              ⏮
            </button>

            {/* Play/Pause */}
            <button className="ctrl-btn" onClick={handlePlayPause} title={isPlaying ? '暂停' : '播放'}>
              {isPlaying ? '⏸' : '▶'}
            </button>

            {/* Next */}
            <button
              className="ctrl-btn ctrl-sm"
              onClick={handleNext}
              disabled={!hasPlaylist}
              title="下一集"
            >
              ⏭
            </button>
          </div>

          {/* Play mode buttons */}
          <div className="control-buttons control-buttons-sm">
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

          {/* Volume + Fullscreen */}
          <div className="bottom-controls">
            <div className="volume-control">
              <button className="ctrl-icon-btn" onClick={handleMuteToggle} title={isMuted ? '取消静音' : '静音'}>
                {isMuted || volume === 0 ? '🔇' : volume < 0.5 ? '🔉' : '🔊'}
              </button>
              <input
                type="range"
                min={0}
                max={1}
                step={0.01}
                value={isMuted ? 0 : volume}
                onChange={handleVolumeChange}
                className="volume-bar"
              />
            </div>

            {isVideo && (
              <button className="ctrl-icon-btn" onClick={handleFullscreen} title="全屏">
                ⛶
              </button>
            )}

            {hasPlaylist && (
              <span className="playlist-info">
                {currentIndex + 1} / {playlistItems.length}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
