package louis

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	ctx           *AppContext
	appRouter     *mux.Router
	metricsRouter *mux.Router
}

func (s *Server) MetricsRouter() *mux.Router {
	return s.metricsRouter
}

func (s *Server) AppRouter() *mux.Router {
	return s.appRouter
}

func NewServer(ctx *AppContext) *Server {
	var s = &Server{
		appRouter:     mux.NewRouter(),
		metricsRouter: mux.NewRouter(),
		ctx:           ctx,
	}
	s.initRoutes()
	return s
}

func (s *Server) initRoutes() {

	var throttler = NewThrottler(s.ctx.Config)

	s.appRouter.Use(addAccessControlAllowOriginHeader(s.ctx.Config))
	s.appRouter.Use(corsMiddleware())
	s.appRouter.Use(recoverFromPanic)

	s.appRouter.HandleFunc("/", handleDashboard).Methods("GET")

	// s.appRouter.Handle("/upload", throttler.Throttle(UploadHandler(s.ctx))).Methods("POST")
	s.appRouter.Handle("/upload",
		throttler.Throttle(
			withSession(s.ctx)(
				authorize(s.ctx.Config.PublicKey)(
					validate()(handleUpload)))),
	).Methods("POST")

	s.appRouter.Handle("/uploadWithClaim",
		throttler.Throttle(
			withSession(s.ctx)(
				authorize(s.ctx.Config.SecretKey)(
					validate()(handleUploadWithClaim)))),
	).Methods("POST")

	s.appRouter.HandleFunc("/claim",
		withSession(s.ctx)(
			authorize(s.ctx.Config.SecretKey)(handleClaim),
		)).Methods("POST")

	s.appRouter.HandleFunc("/healthz", handleHealth).Methods("GET")

	s.metricsRouter.Handle("/metrics", promhttp.Handler())
	s.metricsRouter.HandleFunc("/free", handleFree).Methods("POST")
}
