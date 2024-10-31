.PHONY: server
server:
	@go run *.go --listen :8088 --apikey $$(agepass head nearmap/prod-apikey)

.PHONY: open
open:
	@open http://localhost:8088
