package common

// SQLiteOnNetworkedFilesystem records whether, at startup, Pocket ID detected the SQLite database directory to be on a networked filesystem such as NFS, SMB, or FUSE.
// SQLite locking is unreliable on such filesystems, which can lead to database corruption. It's always false when using Postgres.
var SQLiteOnNetworkedFilesystem bool

// ShowSQLiteStorageWarning reports whether the SQLite-on-networked-filesystem warning should be surfaced to admins in the UI.
func ShowSQLiteStorageWarning() bool {
	return SQLiteOnNetworkedFilesystem && !bool(EnvConfig.DismissSQLiteStorageWarning)
}
