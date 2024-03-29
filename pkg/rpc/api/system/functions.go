package nestedServiceSystem

import (
	"encoding/json"
	"git.ronaksoft.com/nested/server/pkg/config"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"runtime"
	"strings"

	"git.ronaksoft.com/nested/server/nested"
)

// @Command: system/get_counters
func (s *SystemService) getSystemCounters(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	counters := s.Worker().Model().System.GetCounters()
	r := tools.M{}
	for key, val := range counters {
		r[key] = val
	}
	response.OkWithData(r)
}

// @Command: system/get_int_constants
func (s *SystemService) getSystemIntegerConstants(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	m := tools.M{}
	for k, v := range s.Worker().Model().System.GetIntegerConstants() {
		m[k] = v
	}
	response.OkWithData(m)
}

// @Command: system/get_string_constants
func (s *SystemService) getSystemStringConstants(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	m := tools.M{}
	for k, v := range s.Worker().Model().System.GetStringConstants() {
		m[k] = v
	}
	response.OkWithData(m)
}

// @Command: system/set_int_constants
// @Input:  cache_lifetime					int		+
// @Input:  post_max_targets					int		+
// @Input:  post_max_attachments				int		+
// @Input:  post_retract_time	                 int		+
// @Input:  post_max_labels                    int     +
// @Input:  account_grandplaces_limit			int		+
// @Input:  label_max_members           int     +
// @Input:  place_max_children				int		+
// @Input:  place_max_creators				int		+
// @Input:  place_max_keyholders				int		+
// @Input:  register_mode					    int		+	(1: everyone, 2: admin_only)
func (s *SystemService) setSystemIntegerConstants(_ *nested.Account, request *rpc.Request, response *rpc.Response) {
	if len(request.Data) > global.DefaultMaxResultLimit {
		response.Error(global.ErrLimit, []string{"too many parameters"})
		return
	}
	s.Worker().Model().System.SetIntegerConstants(request.Data)
	response.Ok()
}

// @Command: system/set_string_constants
// @Input:  company_name			      string		+
// @Input:  company_desc			      string		+
// @Input:  company_logo			      string		+
// @Input:  system_lang                 string     +
// @Input:  magic_number                string     +
// @Input:  license_key                 string      +
func (s *SystemService) setSystemStringConstants(_ *nested.Account, request *rpc.Request, response *rpc.Response) {
	if len(request.Data) > global.DefaultMaxResultLimit {
		response.Error(global.ErrLimit, []string{"too many parameters"})
		return
	}
	if v, ok := request.Data["magic_number"].(string); ok {
		v = strings.TrimLeft(v, " +0")
		request.Data["magic_number"] = v
	}
	s.Worker().Model().System.SetStringConstants(request.Data)
	response.Ok()
}

// @Command: system/mon_enable
func (s *SystemService) enableMonitor(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	runtime.SetBlockProfileRate(1000000)
	runtime.SetCPUProfileRate(10)
	response.Ok()
}

// @Command: system/mon_disable
func (s *SystemService) disableMonitor(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	runtime.SetBlockProfileRate(0)
	runtime.SetCPUProfileRate(0)
	response.Ok()
}

// @Command: system/stats
func (s *SystemService) getSystemStats(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	M := tools.M{
		nested.SysInfoUserAPI: tools.M{},
		nested.SysInfoGateway: tools.M{},
		nested.SysInfoMsgAPI:  tools.M{},
		nested.SysInfoStorage: tools.M{},
		nested.SysInfoRouter:  tools.M{},
	}

	for key := range M {
		sysInfo := s.Worker().Model().System.GetSystemInfo(key)
		subMap := tools.M{}
		for subKey, subVal := range sysInfo {
			m := tools.M{}
			json.Unmarshal([]byte(subVal), &m)
			subMap[subKey] = m
		}
		M[key] = subMap
	}

	response.OkWithData(M)
}

// @Command: system/mon_activity
// @Input:	mon_access_token				string		*
func (s *SystemService) monitorActivity(_ *nested.Account, request *rpc.Request, response *rpc.Response) {
	if v, ok := request.Data["mon_access_token"].(string); ok {
		if v != config.GetString(config.MonitorAccessToken) {
			response.Error(global.ErrInvalid, []string{"mon_access_token"})
			return
		}
	} else {
		response.Error(global.ErrAccess, []string{""})
		return
	}
	response.OkWithData(tools.M{
		"apis": s.Worker().Model().Report.GetAPICounters(),
	})
}

// @Command: system/online_users
func (s *SystemService) onlineUsers(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	bundleIDs := s.Worker().Model().GetBundles()
	r := make([]tools.M, 0, len(bundleIDs))
	for _, bundleID := range bundleIDs {
		r = append(r, tools.M{
			"bundle_id": bundleID,
			"accounts":  s.Worker().Pusher().GetOnlineAccounts(bundleID),
		})
	}
	response.OkWithData(tools.M{
		"online_users": r,
	})
}

// @Command: system/set_license
// @Input: license_key      string      *
func (s *SystemService) setLicense(_ *nested.Account, request *rpc.Request, response *rpc.Response) {
	var licenseKey string
	if v, ok := request.Data["license_key"].(string); ok {
		licenseKey = v
	} else {
		response.Error(global.ErrIncomplete, []string{"license_key"})
		return
	}
	s.Worker().Model().License.Set(licenseKey)
	if !s.Worker().Model().License.Load() {
		response.Error(global.ErrInvalid, []string{"license_key"})
		return
	}
	if s.Worker().Model().License.IsExpired() {
		response.Error(global.ErrInvalid, []string{"license_key"})
	} else {
		s.Worker().ResetLicense()
		response.Ok()
	}
}

// @Command: system/get_license
func (s *SystemService) getLicense(_ *nested.Account, _ *rpc.Request, response *rpc.Response) {
	license := s.Worker().Model().License.Get()
	response.OkWithData(tools.M{"license": license})
}
