package config

import (
	"time"
)

const (
	PathHealthCheck          = "/"
	PathCreateTag            = "/create_tag"
	PathGetTags              = "/get_tags"
	PathGetTag               = "/get_tag"
	PathCountTags            = "/count_tags"
	PathCreateSegment        = "/create_segment"
	PathGetSegment           = "/get_segment"
	PathGetSegments          = "/get_segments"
	PathCountUd              = "/count_ud"
	PathDownloadUds          = "/download_uds"
	PathPreviewUd            = "/preview_ud"
	PathCountSegments        = "/count_segments"
	PathCreateEmail          = "/create_email"
	PathGetEmails            = "/get_emails"
	PathGetEmail             = "/get_email"
	PathCreateCampaign       = "/create_campaign"
	PathOnEmailAction        = "/on_email_action"
	PathGetCampaigns         = "/get_campaigns"
	PathGetCampaign          = "/get_campaign"
	PathCreateTenant         = "/create_tenant"
	PathGetTenant            = "/get_tenant"
	PathInitUser             = "/init_user"
	PathLogIn                = "/log_in"
	PathLogOut               = "/log_out"
	PathIsLoggedIn           = "/is_logged_in"
	PathCreateFileUploadTask = "/create_file_upload_task"
	PathGetFileUploadTasks   = "/get_file_upload_tasks"
	PathCreateTrialAccount   = "/create_trial_account"
	PathGetActions           = "/get_actions"
	PathCreateRole           = "/create_role"
	PathUpdateRoles          = "/update_roles"
	PathGetRoles             = "/get_roles"
	PathCreateUsers          = "/create_users"
	PathGetUsers             = "/get_users"
	PathMe                   = "/me"
	PathGetDistinctTagValues = "/get_distinct_tag_values"
)

const (
	DefaultPort   = 8080
	LogLevelDebug = "DEBUG"
)

const ThreeMonths = 24 * 30 * 3 * time.Hour
