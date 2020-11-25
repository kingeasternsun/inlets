module inlets

go 1.15

replace github.com/kingeastern/inlets/util => ./util

replace github.com/kingeastern/inlets/types => ./types

replace github.com/kingeastern/inlets/client => ./client

replace github.com/kingeastern/inlets/servermc => ./servermc

replace github.com/kingeastern/inlets/server => ./server

replace github.com/kingeastern/inlets/transport => ./transport

require (
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/kingeastern/inlets/client v0.0.0-00010101000000-000000000000
	github.com/kingeastern/inlets/servermc v0.0.0-00010101000000-000000000000
	github.com/kingeastern/inlets/transport v0.0.0-00010101000000-000000000000 // indirect
	github.com/kingeastern/inlets/types v0.0.0-00010101000000-000000000000
	github.com/kingeastern/inlets/util v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.7.0
	github.com/twinj/uuid v1.0.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)
