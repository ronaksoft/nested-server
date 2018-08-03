package nestedServiceReport

import (
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server-gateway/client"
)

// @Command: report/get_ts_single_val
// @CommandInfo:	returns a set of time-value pairs based on input filters
// @Input:	from		string	+	(YYYY-MM-DD:HH)
// @Input:	to			string 	+	(YYYY-MM-DD:HH)
// @Input:	res			string	+	(h | d | m)
// @Input:   key         string     *
func (s *ReportService) ReportTimeSeriesSingleValue(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
        case nested.REPORT_RESOLUTION_HOUR, nested.REPORT_RESOLUTION_DAY, nested.REPORT_RESOLUTION_MONTH:
            res = v
        default:
            response.Error(nested.ERR_INVALID, []string{"res"})
            return
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"res"})
        return
    }
    if v, ok := request.Data["key"].(string); ok {
        key = v
    }
    result := _Model.Report.GetTimeSeriesSingleValue(from, to, key, res)
    response.OkWithData(nested.M{"result": result})
}
