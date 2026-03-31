package cores

import "gorm.io/gorm"

// ConfigContext provides shared dependencies for config generation.
// All config generators accept this instead of raw *gorm.DB,
// allowing future extensions without breaking signatures.
type ConfigContext struct {
	DB      *gorm.DB
	WarpDir string // path to WARP account data (e.g., /data/warp)
	GeoDir  string // path to GeoIP/GeoSite databases (e.g., /data/geo)
}
