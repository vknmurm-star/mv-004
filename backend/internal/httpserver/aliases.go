package httpserver

import (
	"net/url"

	authrepo "github.com/timemachine-auto/timemachine/internal/domain/auth"
	bookingrepo "github.com/timemachine-auto/timemachine/internal/domain/booking"
	faqrepo "github.com/timemachine-auto/timemachine/internal/domain/faq"
	kbrepo "github.com/timemachine-auto/timemachine/internal/domain/knowledge"
	svcrepo "github.com/timemachine-auto/timemachine/internal/domain/service"
)

// Local alias declarations to keep handler functions readable while still
// wiring to the real domain inputs (pointers stay assignable).
type (
	serviceRepoInput      = svcrepo.ServiceInput
	serviceRepoListParams = svcrepo.ServiceListParams
	bookingRepoListParams = bookingrepo.ListParams
	loginRepoInput        = authrepo.LoginInput
	faqRepoInput          = faqrepo.Input
	knowledgeRepoInput    = kbrepo.Input
	urlQuery              = url.Values
)

func toFaqRepoInput(in faqInput) *faqRepoInput {
	return &faqRepoInput{
		Question:    in.Question,
		Answer:      in.Answer,
		Category:    in.Category,
		SortOrder:   in.SortOrder,
		IsPublished: in.IsPublished,
	}
}
