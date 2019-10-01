package louis

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	AppServer     *http.Server
	MetricsServer *http.Server
	ctx           *AppContext
	appRouter     *mux.Router
	metricsRouter *mux.Router
}

func (s *Server) MetricsRouter() *mux.Router {
	return s.metricsRouter
}

func (s *Server) AppRouter() http.Handler {
	return addAccessControlAllowOriginHeader(s.ctx.Config)(corsMiddleware()(s.appRouter))
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

func (s *Server) Shutdown(ctx context.Context) error {
	var err = s.AppServer.Shutdown(ctx)
	if err != nil {
		return err
	}
	return s.MetricsServer.Shutdown(ctx)
}

func (s *Server) initRoutes() {

	var throttler = NewThrottler(s.ctx.Config)

	// NOTE: this shit does not work, see - https://github.com/gorilla/handlers/issues/142
	// s.appRouter.Use(addAccessControlAllowOriginHeader(s.ctx.Config))
	// s.appRouter.Use(corsMiddleware())
	s.appRouter.Use(recoverFromPanic)

	s.appRouter.HandleFunc("/", handleDashboard).Methods("GET")

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

	s.appRouter.Handle("/restore/{imageKey}",
		throttler.Throttle(
			withSession(s.ctx)(
				authorize(s.ctx.Config.SecretKey)(handleRestore))),
	).Methods("POST")

	s.appRouter.HandleFunc("/healthz", handleHealth).Methods("GET")

	s.metricsRouter.Handle("/metrics", promhttp.Handler())
	s.metricsRouter.HandleFunc("/free", handleFree).Methods("POST")

	s.AppServer = &http.Server{
		Addr:    ":8000", // TODO: get from config
		Handler: s.AppRouter(),
	}

	s.MetricsServer = &http.Server{
		Addr:    ":8001", // TODO: get from config
		Handler: s.MetricsRouter(),
	}

}
