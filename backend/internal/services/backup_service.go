package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

// BackupService manages backup and restore operations
type BackupService struct {
	db              *gorm.DB
	settingsService *SettingsService
	backupDir       string
	dataDir         string
	encryptionKey   []byte
}

// BackupRequest represents a backup creation request
type BackupRequest struct {
	Type              models.BackupType `json:"type"`
	EncryptionEnabled bool              `json:"encryption_enabled"`
	IncludeCores      bool              `json:"include_cores"`
	IncludeCerts      bool              `json:"include_certs"`
	IncludeWARP       bool              `json:"include_warp"`
	IncludeGeo        bool              `json:"include_geo"`
}

// RestoreRequest represents a restore request
type RestoreRequest struct {
	BackupID uint `json:"backup_id"`
	Force    bool `json:"force"`
}

// NewBackupService creates a new backup service
func NewBackupService(db *gorm.DB, settingsService *SettingsService, backupDir string, dataDir string) *BackupService {
	return &BackupService{
		db:              db,
		settingsService: settingsService,
		backupDir:       backupDir,
		dataDir:         dataDir,
	}
}

// Initialize initializes the backup service
func (s *BackupService) Initialize() error {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(s.backupDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Load or generate encryption key
	if err := s.loadOrGenerateEncryptionKey(); err != nil {
		return fmt.Errorf("failed to initialize encryption key: %w", err)
	}

	return nil
}

// loadOrGenerateEncryptionKey loads existing key or generates new one
func (s *BackupService) loadOrGenerateEncryptionKey() error {
	keyPath := filepath.Join(s.dataDir, ".backup_key")

	// Try to load existing key
	data, err := os.ReadFile(keyPath)
	if err == nil {
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err == nil && len(decoded) == 32 {
			s.encryptionKey = decoded
			return nil
		}
	}

	// Generate new key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Save key
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(key)), 0600); err != nil {
		return fmt.Errorf("failed to save encryption key: %w", err)
	}

	s.encryptionKey = key
	return nil
}

// GetEncryptionKey returns the encryption key (for CLI export)
func (s *BackupService) GetEncryptionKey() string {
	return base64.StdEncoding.EncodeToString(s.encryptionKey)
}

// CreateBackup creates a new backup
func (s *BackupService) CreateBackup(req BackupRequest) (*models.Backup, error) {
	startTime := time.Now()

	// Create backup record
	backup := &models.Backup{
		Filename:          fmt.Sprintf("backup_%s.tar.gz", startTime.Format("2006-01-02_15-04-05")),
		FilePath:          filepath.Join(s.backupDir, fmt.Sprintf("backup_%s.tar.gz", startTime.Format("2006-01-02_15-04-05"))),
		BackupType:        req.Type,
		Destination:       models.BackupDestinationLocal,
		Status:            models.BackupStatusRunning,
		EncryptionEnabled: req.EncryptionEnabled,
		CreatedAt:         startTime,
	}

	// Save backup source
	backupSource := models.BackupSource{
		IncludeDatabase: true,
		IncludeCores:    req.IncludeCores,
		IncludeCerts:    req.IncludeCerts,
		IncludeWARP:     req.IncludeWARP,
		IncludeGeo:      req.IncludeGeo,
	}
	sourceJSON, _ := json.Marshal(backupSource)
	backup.BackupSource = string(sourceJSON)

	if err := s.db.Create(backup).Error; err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}

	// Perform backup in goroutine to avoid blocking
	go func() {
		err := s.performBackup(backup, backupSource)

		backup.CompletedAt = func() *time.Time { t := time.Now(); return &t }()
		backup.DurationMs = int(time.Since(startTime).Milliseconds())

		if err != nil {
			backup.Status = models.BackupStatusFailed
			backup.ErrorMessage = err.Error()
		} else {
			backup.Status = models.BackupStatusCompleted
		}

		s.db.Save(backup)

		// Rotate old backups
		if err == nil {
			s.rotateBackups()
		}
	}()

	return backup, nil
}

// performBackup performs the actual backup operation
func (s *BackupService) performBackup(backup *models.Backup, source models.BackupSource) error {
	// Create temporary directory for backup contents
	tmpDir, err := os.MkdirTemp("", "isolate-panel-backup-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Dump database
	if err := s.dumpDatabase(tmpDir); err != nil {
		return fmt.Errorf("failed to dump database: %w", err)
	}

	// 2. Copy core configs
	if source.IncludeCores {
		if err := s.copyCoreConfigs(tmpDir); err != nil {
			return fmt.Errorf("failed to copy core configs: %w", err)
		}
	}

	// 3. Copy certificates
	if source.IncludeCerts {
		if err := s.copyCertificates(tmpDir); err != nil {
			return fmt.Errorf("failed to copy certificates: %w", err)
		}
	}

	// 4. Copy WARP keys
	if source.IncludeWARP {
		if err := s.copyWARPKeys(tmpDir); err != nil {
			return fmt.Errorf("failed to copy WARP keys: %w", err)
		}
	}

	// 5. Copy Geo databases
	if source.IncludeGeo {
		if err := s.copyGeoDatabases(tmpDir); err != nil {
			return fmt.Errorf("failed to copy Geo databases: %w", err)
		}
	}

	// 6. Copy encryption key
	if err := s.copyEncryptionKey(tmpDir); err != nil {
		return fmt.Errorf("failed to copy encryption key: %w", err)
	}

	// 7. Create metadata
	if err := s.createMetadata(tmpDir, backup); err != nil {
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	// 8. Create tar.gz archive
	tarPath := backup.FilePath
	if err := s.createTarArchive(tmpDir, tarPath); err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	// 9. Encrypt if enabled
	if backup.EncryptionEnabled {
		encryptedPath := tarPath + ".enc"
		if err := s.encryptFile(tarPath, encryptedPath); err != nil {
			return fmt.Errorf("failed to encrypt backup: %w", err)
		}
		// Remove unencrypted archive
		os.Remove(tarPath)
		backup.FilePath = encryptedPath
		backup.Filename += ".enc"
	}

	// 10. Calculate checksum
	checksum, err := s.calculateChecksum(backup.FilePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	backup.ChecksumSHA256 = checksum

	// 11. Get file size
	info, err := os.Stat(backup.FilePath)
	if err != nil {
		return fmt.Errorf("failed to get file size: %w", err)
	}
	backup.FileSizeBytes = info.Size()

	// 12. Create metadata file next to backup
	metaPath := strings.TrimSuffix(backup.FilePath, ".enc") + ".meta.json"
	if backup.EncryptionEnabled {
		metaPath = backup.FilePath[:len(backup.FilePath)-4] + ".meta.json"
	}
	if err := s.createMetaFile(metaPath, backup); err != nil {
		return fmt.Errorf("failed to create meta file: %w", err)
	}

	return nil
}

// dumpDatabase creates a SQLite database dump
func (s *BackupService) dumpDatabase(tmpDir string) error {
	dbPath := filepath.Join(s.dataDir, "isolate-panel.db")
	dumpPath := filepath.Join(tmpDir, "database.sql")

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file not found: %s", dbPath)
	}

	// Use sqlite3 .dump command
	cmd := exec.Command("sqlite3", dbPath, ".dump")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sqlite3 dump failed: %w", err)
	}

	return os.WriteFile(dumpPath, out.Bytes(), 0644)
}

// copyCoreConfigs copies core configuration files
func (s *BackupService) copyCoreConfigs(tmpDir string) error {
	coresDir := filepath.Join(tmpDir, "cores")
	if err := os.MkdirAll(coresDir, 0755); err != nil {
		return err
	}

	coreConfigs := []string{
		"cores/xray/config.json",
		"cores/singbox/config.json",
		"cores/mihomo/config.yaml",
	}

	for _, config := range coreConfigs {
		src := filepath.Join(s.dataDir, config)
		dst := filepath.Join(coresDir, filepath.Base(config))

		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dst, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyCertificates copies TLS certificates
func (s *BackupService) copyCertificates(tmpDir string) error {
	certsDir := filepath.Join(s.dataDir, "certs")
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		return nil // No certs to copy
	}

	dstCertsDir := filepath.Join(tmpDir, "certs")
	return s.copyDir(certsDir, dstCertsDir)
}

// copyWARPKeys copies WARP account keys
func (s *BackupService) copyWARPKeys(tmpDir string) error {
	warpDir := filepath.Join(s.dataDir, "warp")
	if _, err := os.Stat(warpDir); os.IsNotExist(err) {
		return nil // No WARP keys to copy
	}

	dstWarpDir := filepath.Join(tmpDir, "warp")
	return s.copyDir(warpDir, dstWarpDir)
}

// copyGeoDatabases copies GeoIP/GeoSite databases
func (s *BackupService) copyGeoDatabases(tmpDir string) error {
	geoDir := filepath.Join(s.dataDir, "geo")
	if _, err := os.Stat(geoDir); os.IsNotExist(err) {
		return nil // No Geo databases to copy
	}

	dstGeoDir := filepath.Join(tmpDir, "geo")
	return s.copyDir(geoDir, dstGeoDir)
}

// copyEncryptionKey copies the encryption key
func (s *BackupService) copyEncryptionKey(tmpDir string) error {
	keyPath := filepath.Join(s.dataDir, ".backup_key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil // No key to copy
	}

	dstPath := filepath.Join(tmpDir, ".backup_key")
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, data, 0600)
}

// createMetadata creates backup metadata file
func (s *BackupService) createMetadata(tmpDir string, backup *models.Backup) error {
	// Get database migration version
	var migrationVersion string
	row := s.db.Raw("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Row()
	if row != nil {
		row.Scan(&migrationVersion)
	}
	if migrationVersion == "" {
		migrationVersion = "unknown"
	}

	// Get hostname
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	metadata := models.BackupMetadata{
		Version:             "1.0",
		IsolatePanelVersion: "0.2.0", // TODO: get from build info
		DatabaseMigration:   migrationVersion,
		CoresIncluded:       []string{"xray", "singbox", "mihomo"},
		Hostname:            hostname,
		CreatedAt:           backup.CreatedAt.Format(time.RFC3339),
	}

	metaJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(tmpDir, "metadata.json"), metaJSON, 0644)
}

// createTarArchive creates a tar.gz archive from directory
func (s *BackupService) createTarArchive(srcDir, dstPath string) error {
	tarGzFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	// Add GZIP compression
	gzipWriter := gzip.NewWriter(tarGzFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.Open(path)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tarWriter, data); err != nil {
				return err
			}
		}

		return nil
	})
}

// encryptFile encrypts a file using chunked AES-256-GCM (streaming support)
func (s *BackupService) encryptFile(srcPath, dstPath string) error {
	const (
		chunkSize = 64 * 1024 // 64KB chunks
		nonceSize = 12
	)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return err
	}

	aesGcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	buffer := make([]byte, chunkSize)
	for {
		n, err := srcFile.Read(buffer)
		if n > 0 {
			nonce := make([]byte, nonceSize)
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
				return err
			}

			// Seal chunk: [nonce][sealed_chunk (ciphertext + tag)]
			sealed := aesGcm.Seal(nil, nonce, buffer[:n], nil)

			// Write chunk length (uint32) to support variable chunk size (if last chunk < chunkSize)
			if err := binary.Write(dstFile, binary.LittleEndian, uint32(len(sealed))); err != nil {
				return err
			}
			if _, err := dstFile.Write(nonce); err != nil {
				return err
			}
			if _, err := dstFile.Write(sealed); err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// decryptFile decrypts a file using chunked AES-256-GCM (streaming support)
func (s *BackupService) decryptFile(srcPath, dstPath string) error {
	const nonceSize = 12

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return err
	}

	aesGcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	for {
		var sealedLen uint32
		err := binary.Read(srcFile, binary.LittleEndian, &sealedLen)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		nonce := make([]byte, nonceSize)
		if _, err := io.ReadFull(srcFile, nonce); err != nil {
			return err
		}

		sealed := make([]byte, sealedLen)
		if _, err := io.ReadFull(srcFile, sealed); err != nil {
			return err
		}

		decrypted, err := aesGcm.Open(nil, nonce, sealed, nil)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}

		if _, err := dstFile.Write(decrypted); err != nil {
			return err
		}
	}

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *BackupService) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// createMetaFile creates metadata file next to backup
func (s *BackupService) createMetaFile(metaPath string, backup *models.Backup) error {
	meta := map[string]interface{}{
		"filename":              backup.Filename,
		"checksum_sha256":       backup.ChecksumSHA256,
		"encrypted_size":        backup.FileSizeBytes,
		"created_at":            backup.CreatedAt.Format(time.RFC3339),
		"backup_version":        "1.0",
		"isolate_panel_version": "0.2.0",
		"database_migration":    "000025",
		"cores_included":        []string{"xray", "singbox", "mihomo"},
		"hostname":              "unknown",
		"encryption_enabled":    backup.EncryptionEnabled,
	}

	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, metaJSON, 0644)
}

// rotateBackups removes old backups keeping only the configured count
func (s *BackupService) rotateBackups() error {
	// Get retention count from settings
	retentionCount := 3 // Default
	if s.settingsService != nil {
		val, err := s.settingsService.GetSettingValue("backup_retention_count")
		if err == nil && val != "" {
			var count int
			fmt.Sscanf(val, "%d", &count)
			if count > 0 {
				retentionCount = count
			}
		}
	}

	var backups []models.Backup
	if err := s.db.Order("created_at DESC").Find(&backups).Error; err != nil {
		return err
	}

	// Keep only configured most recent backups
	if len(backups) > retentionCount {
		for i := retentionCount; i < len(backups); i++ {
			// Delete file
			os.Remove(backups[i].FilePath)
			os.Remove(strings.TrimSuffix(backups[i].FilePath, ".enc") + ".meta.json")

			// Delete record
			s.db.Delete(&backups[i])
		}
	}

	return nil
}

// ListBackups returns list of all backups
func (s *BackupService) ListBackups() ([]models.Backup, error) {
	var backups []models.Backup
	err := s.db.Order("created_at DESC").Find(&backups).Error
	return backups, err
}

// GetBackup returns a single backup by ID
func (s *BackupService) GetBackup(id uint) (*models.Backup, error) {
	var backup models.Backup
	err := s.db.First(&backup, id).Error
	return &backup, err
}

// DeleteBackup deletes a backup
func (s *BackupService) DeleteBackup(id uint) error {
	backup, err := s.GetBackup(id)
	if err != nil {
		return err
	}

	// Delete files
	os.Remove(backup.FilePath)
	os.Remove(strings.TrimSuffix(backup.FilePath, ".enc") + ".meta.json")

	// Delete record
	return s.db.Delete(&models.Backup{}, id).Error
}

// RestoreBackup restores from a backup
func (s *BackupService) RestoreBackup(id uint, force bool) error {
	backup, err := s.GetBackup(id)
	if err != nil {
		return err
	}

	if backup.Status != models.BackupStatusCompleted {
		return fmt.Errorf("backup status is %s, not completed", backup.Status)
	}

	// Update status
	backup.Status = models.BackupStatusRestoring
	s.db.Save(backup)

	// Perform restore in goroutine
	go func() {
		err := s.performRestore(backup)

		if err != nil {
			backup.Status = models.BackupStatusFailed
			backup.ErrorMessage = err.Error()
		} else {
			backup.Status = models.BackupStatusCompleted
		}
		s.db.Save(backup)
	}()

	return nil
}

// performRestore performs the actual restore operation
func (s *BackupService) performRestore(backup *models.Backup) error {
	// Create temporary directory for extraction
	tmpDir, err := os.MkdirTemp("", "isolate-panel-restore-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Decrypt if needed
	backupPath := backup.FilePath
	if strings.HasSuffix(backupPath, ".enc") {
		decryptedPath := filepath.Join(tmpDir, "backup.tar.gz")
		if err := s.decryptFile(backupPath, decryptedPath); err != nil {
			return err
		}
		backupPath = decryptedPath
	}

	// Extract archive
	if err := s.extractTarArchive(backupPath, tmpDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// 1. Restore database
	if err := s.restoreDatabase(tmpDir); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	// 2. Restore core configs
	if err := s.restoreCoreConfigs(tmpDir); err != nil {
		return fmt.Errorf("failed to restore core configs: %w", err)
	}

	// 3. Restore certificates
	if err := s.restoreCertificates(tmpDir); err != nil {
		return fmt.Errorf("failed to restore certificates: %w", err)
	}

	// 4. Restore WARP keys
	if err := s.restoreWARPKeys(tmpDir); err != nil {
		return fmt.Errorf("failed to restore WARP keys: %w", err)
	}

	// 5. Restore Geo databases
	if err := s.restoreGeoDatabases(tmpDir); err != nil {
		return fmt.Errorf("failed to restore Geo databases: %w", err)
	}

	return nil
}

// extractTarArchive extracts a tar.gz archive
func (s *BackupService) extractTarArchive(srcPath, dstDir string) error {
	tarGzFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer tarGzFile.Close()

	// Use GZIP reader
	gzipReader, err := gzip.NewReader(tarGzFile)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dstDir, header.Name)

		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		} else {
			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			targetFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(targetFile, tarReader); err != nil {
				targetFile.Close()
				return err
			}
			targetFile.Close()
		}
	}

	return nil
}

// restoreDatabase restores the database from dump
func (s *BackupService) restoreDatabase(tmpDir string) error {
	dumpPath := filepath.Join(tmpDir, "database.sql")
	dbPath := filepath.Join(s.dataDir, "isolate-panel.db")

	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return nil // No database dump to restore
	}

	// Backup current database
	if _, err := os.Stat(dbPath); err == nil {
		backupPath := dbPath + ".backup." + time.Now().Format("20060102150405")
		if err := os.Rename(dbPath, backupPath); err != nil {
			return fmt.Errorf("failed to backup current database: %w", err)
		}
	}

	// Read dump file
	dumpData, err := os.ReadFile(dumpPath)
	if err != nil {
		return err
	}

	// Create new database
	cmd := exec.Command("sqlite3", dbPath)
	cmd.Stdin = bytes.NewReader(dumpData)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// restoreCoreConfigs restores core configurations
func (s *BackupService) restoreCoreConfigs(tmpDir string) error {
	coresDir := filepath.Join(tmpDir, "cores")
	if _, err := os.Stat(coresDir); os.IsNotExist(err) {
		return nil
	}

	configFiles := []string{"config.json", "config.yaml"}
	for _, configFile := range configFiles {
		src := filepath.Join(coresDir, configFile)
		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}

			// Determine target directory based on config type
			var targetDir string
			if configFile == "config.yaml" {
				targetDir = filepath.Join(s.dataDir, "cores/mihomo")
			} else {
				// Could be xray or singbox - check content or try both
				targetDir = filepath.Join(s.dataDir, "cores/xray")
			}

			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return err
			}

			if err := os.WriteFile(filepath.Join(targetDir, configFile), data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// restoreCertificates restores TLS certificates
func (s *BackupService) restoreCertificates(tmpDir string) error {
	srcCertsDir := filepath.Join(tmpDir, "certs")
	if _, err := os.Stat(srcCertsDir); os.IsNotExist(err) {
		return nil
	}

	dstCertsDir := filepath.Join(s.dataDir, "certs")
	return s.copyDir(srcCertsDir, dstCertsDir)
}

// restoreWARPKeys restores WARP account keys
func (s *BackupService) restoreWARPKeys(tmpDir string) error {
	srcWarpDir := filepath.Join(tmpDir, "warp")
	if _, err := os.Stat(srcWarpDir); os.IsNotExist(err) {
		return nil
	}

	dstWarpDir := filepath.Join(s.dataDir, "warp")
	return s.copyDir(srcWarpDir, dstWarpDir)
}

// restoreGeoDatabases restores GeoIP/GeoSite databases
func (s *BackupService) restoreGeoDatabases(tmpDir string) error {
	srcGeoDir := filepath.Join(tmpDir, "geo")
	if _, err := os.Stat(srcGeoDir); os.IsNotExist(err) {
		return nil
	}

	dstGeoDir := filepath.Join(s.dataDir, "geo")
	return s.copyDir(srcGeoDir, dstGeoDir)
}

// copyDir copies a directory recursively
func (s *BackupService) copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(dstPath, data, info.Mode())
	})
}

// GetSchedule returns the current backup schedule
func (s *BackupService) GetSchedule() (string, error) {
	var backup models.Backup
	err := s.db.Where("schedule_cron IS NOT NULL AND schedule_cron != ''").
		Order("created_at DESC").
		First(&backup).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return backup.ScheduleCron, nil
}

// SetSchedule sets the backup schedule
func (s *BackupService) SetSchedule(cronExpr string) error {
	// Clear existing schedules
	s.db.Model(&models.Backup{}).Update("schedule_cron", "")

	// Set new schedule on latest backup record
	var backup models.Backup
	if err := s.db.Order("created_at DESC").First(&backup).Error; err != nil {
		// Create a new record if none exists
		backup = models.Backup{
			ScheduleCron: cronExpr,
			Status:       models.BackupStatusPending,
		}
		return s.db.Create(&backup).Error
	}

	backup.ScheduleCron = cronExpr
	return s.db.Save(&backup).Error
}

// writeCounter tracks bytes written
type writeCounter struct {
	Total int64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += int64(n)
	return n, nil
}

// DownloadBackup returns backup file as bytes for download
func (s *BackupService) DownloadBackup(id uint) ([]byte, string, error) {
	backup, err := s.GetBackup(id)
	if err != nil {
		return nil, "", err
	}

	data, err := os.ReadFile(backup.FilePath)
	if err != nil {
		return nil, "", err
	}

	return data, backup.Filename, nil
}
