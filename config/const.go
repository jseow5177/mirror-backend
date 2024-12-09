package config

const (
	PathHealthCheck          = "/"
	PathCreateTag            = "/create_tag"
	PathGetTags              = "/get_tags"
	PathCountTags            = "/count_tags"
	PathCreateFileUploadTask = "/create_file_upload_task"
	PathGetMappingIDs        = "/get_mapping_ids"
	PathGetSetMappingIDs     = "/get_set_mapping_ids"
	PathCreateSegment        = "/create_segment"
	PathGetSegments          = "/get_segments"
	PathCountUd              = "/count_ud"
	PathPreviewUd            = "/preview_ud"
	PathCountSegments        = "/count_segments"
	PathCreateEmail          = "/create_email"
	PathGetEmails            = "/get_emails"
	PathCreateCampaign       = "/create_campaign"
	PathOnEmailOpen          = "/on_email_open"
	PathOnEmailButtonClick   = "/on_email_button_click"
	PathRunCampaigns         = "/run_campaigns"
	PathOnEmailAction        = "/on_email_action"
)

const (
	DefaultPort   = 9090
	LogLevelDebug = "DEBUG"
)

var (
	EmptyJson = []byte("{}")
)
