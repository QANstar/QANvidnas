using Microsoft.EntityFrameworkCore;
using QANvidnasServer.Models;

namespace QANvidnasServer.Data;

public class AppDbContext : DbContext
{
    public AppDbContext(DbContextOptions<AppDbContext> options) : base(options) { }

    public DbSet<User> Users => Set<User>();
    public DbSet<InviteCode> InviteCodes => Set<InviteCode>();
    public DbSet<Media> Media => Set<Media>();
    public DbSet<Tag> Tags => Set<Tag>();
    public DbSet<MediaTag> MediaTags => Set<MediaTag>();
    public DbSet<Playlist> Playlists => Set<Playlist>();
    public DbSet<PlaylistItem> PlaylistItems => Set<PlaylistItem>();
    public DbSet<ScanFolder> ScanFolders => Set<ScanFolder>();
    public DbSet<DeviceCode> DeviceCodes => Set<DeviceCode>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        base.OnModelCreating(modelBuilder);

        // User
        modelBuilder.Entity<User>()
            .HasIndex(u => u.Username)
            .IsUnique();

        // MediaTag: composite PK
        modelBuilder.Entity<MediaTag>()
            .HasKey(mt => new { mt.MediaId, mt.TagId });

        modelBuilder.Entity<MediaTag>()
            .HasOne(mt => mt.Media)
            .WithMany(m => m.MediaTags)
            .HasForeignKey(mt => mt.MediaId)
            .OnDelete(DeleteBehavior.Cascade);

        modelBuilder.Entity<MediaTag>()
            .HasOne(mt => mt.Tag)
            .WithMany(t => t.MediaTags)
            .HasForeignKey(mt => mt.TagId)
            .OnDelete(DeleteBehavior.Cascade);

        // PlaylistItem: composite PK
        modelBuilder.Entity<PlaylistItem>()
            .HasKey(pi => new { pi.PlaylistId, pi.MediaId });

        // Tag: unique name
        modelBuilder.Entity<Tag>()
            .HasIndex(t => t.Name)
            .IsUnique();

        // ScanFolder: unique path
        modelBuilder.Entity<ScanFolder>()
            .HasIndex(sf => sf.Path)
            .IsUnique();

        // Media: indexes
        modelBuilder.Entity<Media>()
            .HasIndex(m => m.Type);
        modelBuilder.Entity<Media>()
            .HasIndex(m => m.Deleted);
        modelBuilder.Entity<Media>()
            .HasIndex(m => m.CreatedAt);
    }
}
