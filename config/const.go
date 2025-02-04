package config

const (
	PathHealthCheck          = "/"
	PathCreateTag            = "/create_tag"
	PathGetTags              = "/get_tags"
	PathGetTag               = "/get_tag"
	PathCountTags            = "/count_tags"
	PathGetMappingIDs        = "/get_mapping_ids"
	PathGetSetMappingIDs     = "/get_set_mapping_ids"
	PathCreateSegment        = "/create_segment"
	PathGetSegment           = "/get_segment"
	PathGetSegments          = "/get_segments"
	PathCountUd              = "/count_ud"
	PathPreviewUd            = "/preview_ud"
	PathCountSegments        = "/count_segments"
	PathCreateEmail          = "/create_email"
	PathGetEmails            = "/get_emails"
	PathGetEmail             = "/get_email"
	PathCreateCampaign       = "/create_campaign"
	PathRunCampaigns         = "/run_campaigns"
	PathOnEmailAction        = "/on_email_action"
	PathGetCampaigns         = "/get_campaigns"
	PathGetCampaign          = "/get_campaign"
	PathCreateTenant         = "/create_tenant"
	PathGetTenant            = "/get_tenant"
	PathInitTenant           = "/init_tenant"
	PathIsTenantPendingInit  = "/is_tenant_pending_init"
	PathCreateUser           = "/create_user"
	PathInitUser             = "/init_user"
	PathIsUserPendingInit    = "/is_user_pending_init"
	PathLogIn                = "/log_in"
	PathLogOut               = "/log_out"
	PathIsLoggedIn           = "/is_logged_in"
	PathCreateFileUploadTask = "/create_file_upload_task"
	PathGetFileUploadTasks   = "/get_file_upload_tasks"
)

const (
	DefaultPort   = 9090
	LogLevelDebug = "DEBUG"
)

var (
	EmptyJson = []byte("{}")
)
