package setting

import (
	"gopkg.in/ini.v1"
)

// AnnotationsLokiSettings contains configuration for using Loki as annotations storage
type AnnotationsLokiSettings struct {
	Enabled           bool
	URL               string
	TenantID          string
	BasicAuthUser     string
	BasicAuthPassword string
}

// readAnnotationsLokiSettings reads the annotations Loki configuration
func (cfg *Cfg) readAnnotationsLokiSettings(iniFile *ini.File) {
	section := iniFile.Section("annotations.loki")

	cfg.AnnotationsLoki = AnnotationsLokiSettings{
		Enabled:           section.Key("enabled").MustBool(false),
		URL:               section.Key("url").MustString(""),
		TenantID:          section.Key("tenant_id").MustString(""),
		BasicAuthUser:     section.Key("basic_auth_user").MustString(""),
		BasicAuthPassword: section.Key("basic_auth_password").MustString(""),
	}
}
