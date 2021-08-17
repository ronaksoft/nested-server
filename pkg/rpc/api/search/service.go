package nestedServiceSearch

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix string = "search"
)
const (
	CmdPlacesForCompose       = "search/places_for_compose"
	CmdPlacesForSearch        = "search/places_for_search"
	CmdAccountsForInvite      = "search/accounts_for_invite"
	CmdAccountsForAdd         = "search/accounts_for_add"
	CmdAccountsForMention     = "search/accounts_for_mention"
	CmdAccountsForTaskMention = "search/accounts_for_task_mention"
	CmdAccountsForSearch      = "search/accounts_for_search"
	CmdAccountsForAdmin       = "search/accounts_for_admin"
	CmdAccounts               = "search/accounts"
	CmdSuggestions            = "search/suggestions"
	CmdLabels                 = "search/labels"
	CmdPosts                  = "search/posts"
	CmdPostsConversation      = "search/posts_conversation"
	CmdTasks                  = "search/tasks"
	CmdApps                   = "search/apps"
)

type SearchService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewSearchService(worker *api.Worker) api.Service {
	s := new(SearchService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdAccounts:               {MinAuthLevel: api.AuthLevelAppL1, Execute: s.accounts},
		CmdAccountsForAdmin:       {MinAuthLevel: api.AuthLevelUser, Execute: s.accountsForAdmin},
		CmdAccountsForSearch:      {MinAuthLevel: api.AuthLevelUser, Execute: s.accountsForSearch},
		CmdAccountsForAdd:         {MinAuthLevel: api.AuthLevelUser, Execute: s.accountsForAdd},
		CmdAccountsForInvite:      {MinAuthLevel: api.AuthLevelUser, Execute: s.accountsForInvite},
		CmdAccountsForMention:     {MinAuthLevel: api.AuthLevelUser, Execute: s.accountsForMention},
		CmdAccountsForTaskMention: {MinAuthLevel: api.AuthLevelUser, Execute: s.accountsForTaskMention},
		CmdLabels:                 {MinAuthLevel: api.AuthLevelUser, Execute: s.labels},
		CmdPlacesForCompose:       {MinAuthLevel: api.AuthLevelAppL1, Execute: s.placesForCompose},
		CmdPlacesForSearch:        {MinAuthLevel: api.AuthLevelAppL1, Execute: s.placesForSearch},
		CmdPosts:                  {MinAuthLevel: api.AuthLevelUser, Execute: s.posts},
		CmdPostsConversation:      {MinAuthLevel: api.AuthLevelUser, Execute: s.conversation},
		CmdSuggestions:            {MinAuthLevel: api.AuthLevelUser, Execute: s.suggestions},
		CmdTasks:                  {MinAuthLevel: api.AuthLevelUser, Execute: s.tasks},
		CmdApps:                   {MinAuthLevel: api.AuthLevelUser, Execute: s.apps},
	}

	return s
}

func (s *SearchService) GetServicePrefix() string {
	return ServicePrefix
}

func (s *SearchService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	commandName := request.Command
	if cmd, ok := s.serviceCommands[commandName]; ok {
		if authLevel >= cmd.MinAuthLevel {
			cmd.Execute(requester, request, response)
		} else {
			response.NotAuthorized()
		}
	} else {
		response.NotImplemented()
	}
}

func (s *SearchService) Worker() *api.Worker {
	return s.worker
}
