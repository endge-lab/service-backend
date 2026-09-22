package config

// SnapshotConfig задаёт срок хранения временных страховочных копий импорта.
type SnapshotConfig struct {
	ImportBackupRetentionDays int
}

func loadSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{ImportBackupRetentionDays: envInt("IMPORT_BACKUP_RETENTION_DAYS", 7)}
}
