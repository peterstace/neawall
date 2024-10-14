.PHONY: server
server:
	@go run *.go --listen :8080 --apikey $$(agepass head nearmap/prod-apikey)

.PHONY: open
open:
	@open http://localhost:8080
