package autolink

import "github.com/cli/cli/v2/pkg/cmdutil"

type autolink struct {
	ID             int    `json:"id"`
	IsAlphanumeric bool   `json:"is_alphanumeric"`
	KeyPrefix      string `json:"key_prefix"`
	URLTemplate    string `json:"url_template"`
}

func (s *autolink) ExportData(fields []string) map[string]interface{} {
	return cmdutil.StructExportData(s, fields)
}
