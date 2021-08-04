package nestedServiceSearch

import (
	"git.ronaksoft.com/nested/server/pkg/global"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	tools "git.ronaksoft.com/nested/server/pkg/toolbox"
	"strings"

	"git.ronaksoft.com/nested/server/model"
)

// @Command:	search/places_for_compose
// @Input:	keyword			string		+
// @Pagination
func (s *SearchService) placesForCompose(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	pg := s.Worker().Argument().GetPagination(request)
	places := s.Worker().Model().Search.PlacesForCompose(keyword, requester.ID, pg)

	r := make([]tools.M, 0, len(places))
	for _, p := range places {
		r = append(r, tools.M{
			"_id":     p.ID,
			"name":    p.Name,
			"picture": p.Picture,
		})
	}
	recipients := s.Worker().Model().Search.RecipientsForCompose(keyword, requester.ID, pg)
	response.OkWithData(tools.M{
		"places":     r,
		"recipients": recipients,
	})
}

// @Command:	search/places_for_search
// @Input:	keyword			string		+
// @Pagination
func (s *SearchService) placesForSearch(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	pg := s.Worker().Argument().GetPagination(request)
	places := s.Worker().Model().Search.PlacesForSearch(keyword, requester.ID, pg)
	r := make([]tools.M, 0, len(places))
	for _, p := range places {
		r = append(r, tools.M{
			"_id":     p.ID,
			"name":    p.Name,
			"picture": p.Picture,
		})
	}
	response.OkWithData(tools.M{"places": r})
}

// @Command:	search/accounts
// @Input:	keyword			string		+
// @Pagination
func (s *SearchService) accounts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}

	pg := s.Worker().Argument().GetPagination(request)
	accounts := s.Worker().Model().Search.Accounts(keyword, nested.AccountSearchFilterUsersEnabled, "", pg)
	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})
}

// @Command:	search/accounts_for_admin
// @Input:	keyword			string		+
// @Pagination
func (s *SearchService) accountsForAdmin(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}

	pg := s.Worker().Argument().GetPagination(request)
	accounts := s.Worker().Model().Search.Accounts(keyword, nested.AccountSearchFilterAll, "", pg)
	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})
}

// @Command:	search/accounts_for_invite
// @Input:	keyword			string		+
// @Input:	place_id		string		*
// @Pagination
func (s *SearchService) accountsForInvite(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accounts []nested.Account
	var keyword string
	place := s.Worker().Argument().GetPlace(request, response)

	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}

	pg := s.Worker().Argument().GetPagination(request)
	if place == nil {
		accounts = s.Worker().Model().Search.AccountsForSearch(requester.ID, keyword, pg)
	} else {
		accounts = s.Worker().Model().Search.AccountsForAddToGrandPlace(requester.ID, place.ID, keyword, pg)
	}

	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})
}

// @Command:	search/accounts_for_add
// @Input:	keyword			string		+
// @Input:	place_id			string		*
// @Pagination
func (s *SearchService) accountsForAdd(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var placeID string
	var keywords []string
	if v, ok := request.Data["keyword"].(string); ok {
		keywords = strings.Split(v, " ")
	}
	if v, ok := request.Data["place_id"].(string); ok {
		placeID = v
	}
	pg := s.Worker().Argument().GetPagination(request)
	accounts := s.Worker().Model().Search.AccountsForAddToPlace(requester.ID, placeID, keywords, pg)
	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})
}

// @Command:	search/accounts_for_mention
// @Input:	keyword		string		+
// @Input:	post_id		string		*
func (s *SearchService) accountsForMention(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keywords []string
	var post *nested.Post
	if post = s.Worker().Argument().GetPost(request, response); post == nil {
		return
	}
	if v, ok := request.Data["keyword"].(string); ok {
		keywords = strings.Split(v, " ")
	}
	pg := s.Worker().Argument().GetPagination(request)
	accounts := s.Worker().Model().Search.AccountsForPostMention(post.PlaceIDs, keywords, pg)

	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})

}

// @Command:	search/accounts_for_task_mention
// @Input:	keyword		string		+
// @Input:	task_id		string		*
func (s *SearchService) accountsForTaskMention(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	var task *nested.Task
	if task = s.Worker().Argument().GetTask(request, response); task == nil {
		return
	}
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	accounts := s.Worker().Model().Search.AccountsForTaskMention(task, keyword)

	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})
}

// @Command:	search/accounts_for_search
// @Input:	keyword			string		+
// @Pagination
func (s *SearchService) accountsForSearch(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}

	pg := s.Worker().Argument().GetPagination(request)
	accounts := s.Worker().Model().Search.AccountsForSearch(requester.ID, keyword, pg)
	r := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		r = append(r, s.Worker().Map().Account(a, false))
	}
	response.OkWithData(tools.M{"accounts": r})

}

// @Command:	search/labels
// @Input:	keyword			string	+
// @Input:	filter			string	+	(my_privates | privates | public | all)
// @Input:	details			bool		+
func (s *SearchService) labels(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword, filter string
	var details bool
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	if v, ok := request.Data["filter"].(string); ok {
		filter = v
	}
	if v, ok := request.Data["details"].(bool); ok {
		details = v
	}

	pg := s.Worker().Argument().GetPagination(request)
	switch filter {
	case nested.LabelFilterMyPrivates, nested.LabelFilterPrivates, nested.LabelFilterPublic,
		nested.LabelFilterAll, nested.LabelFilterMyLabels:
	default:
		filter = nested.LabelFilterAll
	}

	// if user is not a label editor and is not asking for his/her labels then details is not allowed
	if !requester.Authority.LabelEditor && filter != nested.LabelFilterMyPrivates {
		details = false
	}

	labels := s.Worker().Model().Search.Labels(requester.ID, keyword, filter, pg)
	r := make([]tools.M, 0, pg.GetLimit())
	for _, label := range labels {
		r = append(r, s.Worker().Map().Label(requester, label, details))
	}
	response.OkWithData(tools.M{
		"skip":   pg.GetSkip(),
		"limit":  pg.GetLimit(),
		"labels": r,
	})
}

// @Command:	search/posts
// @Input:	keyword			string	+
// @Input:	place_id			string 	+	(comma separated)
// @Input:	sender_id		string 	+	(comma separated)
// @Input:	label_id			string 	+	(comma separated)
// @Input:	label_title		string 	+	(comma separated)
// @Input:	has_attachment	 bool	+
// @Pagination
func (s *SearchService) posts(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var hasAttachment bool
	var keyword string
	senderIDs := make([]string, 0, 10)
	labelIDs := make([]string, 0, 10)
	placeIDs := make([]string, 0, 10)
	if v, ok := request.Data["place_id"].(string); ok && len(v) > 0 {
		placeIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	}
	if v, ok := request.Data["sender_id"].(string); ok && len(v) > 0 {
		senderIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	}
	if v, ok := request.Data["label_id"].(string); ok && len(v) > 0 {
		labelIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	}
	if v, ok := request.Data["label_title"].(string); ok && len(v) > 0 {
		labelTitles := strings.SplitN(v, ",", global.DefaultMaxResultLimit)
		labels := s.Worker().Model().Label.GetByTitles(labelTitles)
		for _, label := range labels {
			labelIDs = append(labelIDs, label.ID)
		}
	}
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	if v, ok := request.Data["has_attachment"].(bool); ok && v {
		hasAttachment = true
	}
	if len(placeIDs) == 0 {
		placeIDs = requester.AccessPlaceIDs
	}
	pg := s.Worker().Argument().GetPagination(request)
	posts := s.Worker().Model().Search.Posts(keyword, requester.ID, placeIDs, senderIDs, labelIDs, hasAttachment, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	s.Worker().Model().Search.AddSearchHistory(requester.ID, keyword)
	response.OkWithData(tools.M{
		"skip":    pg.GetSkip(),
		"limit":   pg.GetLimit(),
		"history": s.Worker().Model().Search.GetSearchHistory(requester.ID, keyword),
		"posts":   r,
	})
}

// @Command:	search/tasks
// @Input:	keyword				string	+
// @Input:	assigner_id			string 	+	(comma separated)
// @Input:	assignee_id			string	+ 	(comma separated)
// @Input:	label_id			    string 	+	(comma separated)
// @Input:	label_title			string 	+	(comma separated)
// @Input:	has_attachment	 	bool	    +
func (s *SearchService) tasks(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var hasAttachment bool
	var keyword string
	assignorIDs := make([]string, 0, 10)
	assigneeIDs := make([]string, 0, 10)
	labelIDs := make([]string, 0, 10)
	if v, ok := request.Data["assigner_id"].(string); ok && len(v) > 0 {
		assignorIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	}
	if v, ok := request.Data["assignee_id"].(string); ok && len(v) > 0 {
		assigneeIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	}
	if v, ok := request.Data["label_id"].(string); ok && len(v) > 0 {
		labelIDs = strings.SplitN(v, ",", global.DefaultMaxResultLimit)
	}
	if v, ok := request.Data["label_title"].(string); ok && len(v) > 0 {
		labelTitles := strings.SplitN(v, ",", global.DefaultMaxResultLimit)
		labels := s.Worker().Model().Label.GetByTitles(labelTitles)
		for _, label := range labels {
			labelIDs = append(labelIDs, label.ID)
		}
	}
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	if v, ok := request.Data["has_attachment"].(bool); ok && v {
		hasAttachment = v
	}
	pg := s.Worker().Argument().GetPagination(request)
	tasks := s.Worker().Model().Search.Tasks(keyword, requester.ID, assignorIDs, assigneeIDs, labelIDs, hasAttachment, pg)

	r := make([]tools.M, 0, len(tasks))
	for _, task := range tasks {
		r = append(r, s.Worker().Map().Task(requester, task, true))
	}
	s.Worker().Model().Search.AddSearchHistory(requester.ID, keyword)
	response.OkWithData(tools.M{
		"skip":    pg.GetSkip(),
		"limit":   pg.GetLimit(),
		"history": s.Worker().Model().Search.GetSearchHistory(requester.ID, keyword),
		"tasks":   r,
	})
}

// @Command:	search/apps
// @Input:	keyword				string	+
func (s *SearchService) apps(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	apps := s.Worker().Model().Search.Apps(keyword, s.Worker().Argument().GetPagination(request))
	r := make([]tools.M, 0, len(apps))
	for _, app := range apps {
		r = append(r, s.Worker().Map().App(app))
	}
	response.OkWithData(tools.M{"apps": r})
}

// @Command:	search/posts_conversation
// @Input:	account_id		string	*
// @Input:	keyword			string	+
// @Pagination
func (s *SearchService) conversation(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var accountID, keywords string
	if v, ok := request.Data["account_id"].(string); ok {
		accountID = v
	} else {
		response.Error(global.ErrIncomplete, []string{"account_id"})
		return
	}
	if v, ok := request.Data["keywords"].(string); ok {
		keywords = v
	}
	if v, ok := request.Data["keyword"].(string); ok {
		keywords = v
	}
	pg := s.Worker().Argument().GetPagination(request)
	posts := s.Worker().Model().Search.PostsConversations(requester.ID, accountID, keywords, pg)
	r := make([]tools.M, 0, len(posts))
	for _, post := range posts {
		r = append(r, s.Worker().Map().Post(requester, post, true))
	}
	response.OkWithData(tools.M{
		"skip":  pg.GetSkip(),
		"limit": pg.GetLimit(),
		"posts": r,
	})
}

// @Command:	search/suggestions
// @Input:	keyword			string	+
func (s *SearchService) suggestions(requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	var keyword string
	if v, ok := request.Data["keyword"].(string); ok {
		keyword = v
	}
	pg := s.Worker().Argument().GetPagination(request)
	pg.SetLimit(10)

	// Search Places
	places := s.Worker().Model().Search.PlacesForSearch(keyword, requester.ID, pg)
	rPlaces := make([]tools.M, 0, len(places))
	for _, p := range places {
		rPlaces = append(rPlaces, tools.M{
			"_id":     p.ID,
			"name":    p.Name,
			"picture": p.Picture,
		})
	}

	// Search Accounts
	accounts := s.Worker().Model().Search.AccountsForSearch(requester.ID, keyword, pg)
	rAccounts := make([]tools.M, 0, len(accounts))
	for _, a := range accounts {
		rAccounts = append(rAccounts, s.Worker().Map().Account(a, false))
	}

	// Search Labels
	labels := s.Worker().Model().Search.Labels(requester.ID, keyword, nested.LabelFilterAll, pg)
	rLabels := make([]tools.M, 0, len(labels))
	for _, a := range labels {
		rLabels = append(rLabels, s.Worker().Map().Label(requester, a, false))
	}

	response.OkWithData(tools.M{
		"places":   rPlaces,
		"accounts": rAccounts,
		"labels":   rLabels,
		"history":  s.Worker().Model().Search.GetSearchHistory(requester.ID, keyword),
	})
}
