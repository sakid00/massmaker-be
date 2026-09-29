package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/sakid00/massmaker-be/internal/enmasse"
	"github.com/sakid00/massmaker-be/internal/http/handlers"
	mw "github.com/sakid00/massmaker-be/internal/http/middleware"
	"github.com/sakid00/massmaker-be/internal/media"
	"github.com/sakid00/massmaker-be/internal/service"
)

type RouterOptions struct {
	CORSOrigins     string
	JWTAccessSecret string
	JWTIssuer       string
	Media           *media.Presigner
}

func NewRouter(cat *service.Catalogue, events *service.Events, admin *service.Admin, em *enmasse.Client, self *service.MakerSelf, consents *service.Consents, orders *service.Orders, opts RouterOptions) http.Handler {
	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(mw.CORS(opts.CORSOrigins))

	pub := handlers.NewPublicHandler(cat, events)
	adm := handlers.NewAdminHandler(admin)
	id := handlers.NewIdentityHandler(em)
	meMaker := handlers.NewMeMakerHandler(self)
	mediaH := handlers.NewMediaHandler(opts.Media)
	cons := handlers.NewConsentsHandler(consents)
	ord := handlers.NewOrdersHandler(orders)

	r.Get("/health", handlers.Health)

	r.Route("/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.With(httprate.LimitByIP(10, 10*time.Minute)).Post("/email/status", id.Proxy)
			r.With(httprate.LimitByIP(10, 10*time.Minute)).Post("/register/artist", id.Proxy)
			r.With(httprate.LimitByIP(10, 10*time.Minute)).Post("/register/vendor", id.Proxy)
			r.With(httprate.LimitByIP(5, 10*time.Minute)).Post("/login", id.Proxy)
			r.With(httprate.LimitByIP(5, 10*time.Minute)).Post("/claim", id.Proxy)
			r.With(httprate.LimitByIP(5, 10*time.Minute)).Post("/password/claim", id.Proxy)
			r.With(httprate.LimitByIP(5, 10*time.Minute)).Post("/password/set", id.Proxy)
			r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/password/change", id.Proxy)
		})

		r.Get("/categories", pub.Categories)
		r.Get("/search", pub.Search)
		r.Get("/makers/{id}", pub.Maker)
		r.With(httprate.LimitByIP(30, time.Minute), mw.OptionalHumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/events", pub.Events)
		r.With(httprate.LimitByIP(30, time.Minute), mw.OptionalHumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/consents", cons.Put)

		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/profile", id.Proxy)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Patch("/me/profile", id.Proxy)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/consents", cons.Me)
		r.With(httprate.LimitByIP(10, 10*time.Minute), mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/me/media/presign", mediaH.Presign)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/onboarding", meMaker.Onboarding)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/maker", meMaker.Get)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Patch("/me/maker", meMaker.Patch)
		r.With(httprate.LimitByIP(10, 10*time.Minute), mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/me/category-proposals", meMaker.ProposeCategory)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/category-proposals", meMaker.ListCategoryProposals)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/vendors", pub.MeVendors)
		r.With(httprate.LimitByIP(10, 10*time.Minute), mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/me/inquiry-media/presign", ord.Presign)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/makers/{id}/inquiry-context", ord.Context)
		r.With(httprate.LimitByIP(10, 10*time.Minute), mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Post("/makers/{id}/inquiries", ord.Create)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/orders", ord.List)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Get("/me/orders/{id}", ord.Get)
		r.With(mw.HumanAuth(opts.JWTAccessSecret, opts.JWTIssuer)).Patch("/me/orders/{id}", ord.Patch)

		r.Route("/admin", func(r chi.Router) {
			r.With(httprate.LimitByIP(5, time.Minute)).Post("/login", adm.Login)
			r.Group(func(r chi.Router) {
				r.Use(mw.StaffAuth(admin.ParseStaff))
				r.Get("/makers", adm.ListMakers)
				r.Post("/makers", adm.CreateMaker)
				r.Post("/makers/{id}/publish", adm.Publish)
				r.Post("/makers/{id}/withdraw", adm.Withdraw)
				r.Put("/makers/{id}/contact", adm.PutContact)
				r.Post("/makers/{id}/contact/consent", adm.Consent)
				r.Get("/taxonomy/drafts", adm.ListTaxonomyDrafts)
				r.Post("/category-groups/{slug}/publish", adm.PublishCategoryGroup)
				r.Post("/category-groups/{slug}/withdraw", adm.WithdrawCategoryGroup)
				r.Post("/categories/{slug}/publish", adm.PublishCategory)
				r.Post("/categories/{slug}/withdraw", adm.WithdrawCategory)
			})
		})
	})

	return r
}
