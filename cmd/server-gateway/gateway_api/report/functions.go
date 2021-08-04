package nestedServiceReport

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
)

// @Command: report/get_ts_single_val
// @CommandInfo:	returns a set of time-value pairs based on input filters
// @Input:	from		string	+	(YYYY-MM-DD:HH)
// @Input:	to			string 	+	(YYYY-MM-DD:HH)
// @Input:	res			string	+	(h | d | m)
// @Input:   key         string     *
func (s *ReportService) ReportTimeSeriesSingleValue(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var from, to, key, res string
	if v, ok := request.Data["from"].(string); ok {
		// TODO:: check from format YYYY-MM-DD:HH
		from = v
	}
	if v, ok := request.Data["to"].(string); ok {
		to = v
	}
	if v, ok := request.Data["res"].(string); ok {
		switch v {
		case nested.ReportResolutionHour, nested.ReportResolutionDay, nested.ReportResolutionMonth:
			res = v
		default:
			response.Error(global.ErrInvalid, []string{"res"})
			return
		}
	} else {
		response.Error(global.ErrIncomplete, []string{"res"})
		return
	}
	if v, ok := request.Data["key"].(string); ok {
		key = v
	}
	result := _Model.Report.GetTimeSeriesSingleValue(from, to, key, res)
	response.OkWithData(tools.M{"result": result})
}
